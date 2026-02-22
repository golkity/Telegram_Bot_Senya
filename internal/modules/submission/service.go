package submission

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"telegram_bot/pkg/compressor"
	"telegram_bot/pkg/crypto"
)

type FileUploader interface {
	UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error
	GetPresignedURL(ctx context.Context, objectName string, lifetime time.Duration) (string, error)
	DownloadFile(ctx context.Context, objectName string) (io.ReadCloser, error)
	MoveFile(ctx context.Context, oldKey, newKey string) error
}

type FileDownloader interface {
	GetFileContent(fileID string) (io.ReadCloser, error)
}

type Service struct {
	repo       Repository
	uploader   FileUploader
	downloader FileDownloader
	log        *slog.Logger

	encKey   []byte
	pathSalt []byte
}

func NewService(
	repo Repository,
	uploader FileUploader,
	downloader FileDownloader,
	log *slog.Logger,
	encKey string,
	pathSalt string,
) *Service {
	return &Service{
		repo:       repo,
		uploader:   uploader,
		downloader: downloader,
		log:        log,
		encKey:     []byte(encKey),
		pathSalt:   []byte(pathSalt),
	}
}

func (s *Service) GenerateS3Path(userID int64, taskNumber string) string {
	mac := hmac.New(sha256.New, s.pathSalt)
	mac.Write([]byte(fmt.Sprintf("%d", userID)))
	userHash := hex.EncodeToString(mac.Sum(nil))[:16]

	newFileName := uuid.New().String() + ".bin.enc"
	return fmt.Sprintf("%s/task_%s/%s", userHash, taskNumber, newFileName)
}

func (s *Service) ProcessSubmission(ctx context.Context, input InputDTO) error {
	var s3Paths []string
	var originalNames []string

	courseName := strings.ReplaceAll(input.CourseName, " ", "_")
	curatorName := strings.ReplaceAll(input.CuratorName, " ", "_")
	studentName := strings.ReplaceAll(input.StudentName, " ", "_")

	for _, fileDTO := range input.Files {
		safeFileName := strings.ReplaceAll(fileDTO.FileName, " ", "_")
		s3Path := fmt.Sprintf("%s/%s/%s/Task_%s/%s_%s.bin.enc",
			courseName, curatorName, studentName, input.TaskNumber, uuid.New().String()[:8], safeFileName)

		originalStream, err := s.downloader.GetFileContent(fileDTO.FileID)
		if err != nil {
			return fmt.Errorf("telegram download error: %w", err)
		}

		compressedStream := compressor.CompressStream(originalStream)

		encryptedStream, err := crypto.EncryptStream(compressedStream, s.encKey)
		if err != nil {
			originalStream.Close()
			return fmt.Errorf("encryption setup error: %w", err)
		}

		processedData, err := io.ReadAll(encryptedStream)
		originalStream.Close()
		if err != nil {
			return fmt.Errorf("stream processing error: %w", err)
		}

		err = s.uploader.UploadFile(
			ctx,
			s3Path,
			bytes.NewReader(processedData),
			int64(len(processedData)),
			"application/octet-stream",
		)
		if err != nil {
			return fmt.Errorf("s3 upload error: %w", err)
		}

		s3Paths = append(s3Paths, s3Path)
		originalNames = append(originalNames, fileDTO.FileName)
	}

	sub := Submission{
		UserID:        input.UserID,
		CuratorID:     input.CuratorID,
		Type:          input.Type,
		TaskNumber:    input.TaskNumber,
		FilePaths:     s3Paths,
		OriginalNames: originalNames,
		Comment:       input.Comment,
	}

	return s.repo.Save(ctx, sub)
}

func (s *Service) GetAllSubmissions(ctx context.Context, userID int64) ([]Submission, error) {
	return s.repo.GetAllByUserID(ctx, userID)
}

func (s *Service) GetSubmissionsByTask(ctx context.Context, userID int64, subType Type, taskNum string) ([]Submission, error) {
	return s.repo.GetByTask(ctx, userID, subType, taskNum)
}

func (s *Service) GetSubmissionByID(ctx context.Context, id int64) (*Submission, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetFileLink(ctx context.Context, fileID string) (string, error) {
	return s.uploader.GetPresignedURL(ctx, fileID, 1*time.Hour)
}

func (s *Service) GetSubtasks(ctx context.Context, taskNum string) ([]Subtask, error) {
	if taskNum == "1" {
		return []Subtask{
			{Code: "1.1", Name: "1.1 Графы"},
			{Code: "1.2", Name: "1.2 Таблицы"},
		}, nil
	}
	return nil, nil
}

func (s *Service) GenerateUserArchive(ctx context.Context, userID int64) (string, error) {
	submissions, err := s.repo.GetAllByUserID(ctx, userID)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, sub := range submissions {
		for i, s3Path := range sub.FilePaths {
			originalName := "file"
			if i < len(sub.OriginalNames) {
				originalName = sub.OriginalNames[i]
			}

			zipEntryName := fmt.Sprintf("Task_%s/%s", sub.TaskNumber, originalName)
			s3Stream, err := s.uploader.DownloadFile(ctx, s3Path)

			if err != nil {
				s.log.Error("archive: s3 download failed", "path", s3Path, "err", err)
				continue
			}

			decryptedStream, err := crypto.DecryptStream(s3Stream, s.encKey)
			if err != nil {
				s.log.Error("archive: decryption failed", "err", err)
				s3Stream.Close()
				continue
			}

			decompressedStream, err := compressor.DecompressStream(decryptedStream)
			if err != nil {
				s.log.Error("archive: decompression failed", "err", err)
				s3Stream.Close()
				continue
			}

			writer, err := zipWriter.Create(zipEntryName)
			if err == nil {
				io.Copy(writer, decompressedStream)
			}

			s3Stream.Close()
		}
	}

	if err := zipWriter.Close(); err != nil {
		return "", err
	}

	archivePath := fmt.Sprintf("archives/student_%d/archive_%d.zip", userID, time.Now().Unix())
	if err := s.uploader.UploadFile(ctx, archivePath, bytes.NewReader(buf.Bytes()), int64(buf.Len()), "application/zip"); err != nil {
		return "", err
	}

	return s.uploader.GetPresignedURL(ctx, archivePath, 24*time.Hour)
}

func (s *Service) MigrateStudentFiles(ctx context.Context, studentID int64, newCuratorID int64, newCuratorName string) error {
	submissions, err := s.GetAllSubmissions(ctx, studentID)
	if err != nil {
		return err
	}

	safeNewCurator := strings.ReplaceAll(newCuratorName, " ", "_")

	for _, sub := range submissions {
		var updatedPaths []string
		changed := false

		for _, oldPath := range sub.FilePaths {
			parts := strings.Split(oldPath, "/")
			if len(parts) >= 5 {

				if parts[1] != safeNewCurator {
					parts[1] = safeNewCurator
					newPath := strings.Join(parts, "/")

					err := s.uploader.MoveFile(ctx, oldPath, newPath)
					if err == nil {
						updatedPaths = append(updatedPaths, newPath)
						changed = true
					} else {
						s.log.Error("s3 migration failed", "old", oldPath, "err", err)
						updatedPaths = append(updatedPaths, oldPath)
					}
				} else {
					updatedPaths = append(updatedPaths, oldPath)
				}
			} else {
				updatedPaths = append(updatedPaths, oldPath)
			}
		}

		if changed {
			err = s.repo.UpdatePathsAndCurator(ctx, sub.ID, newCuratorID, updatedPaths)
			if err != nil {
				s.log.Error("db path update failed", "sub_id", sub.ID, "err", err)
			}
		}
	}
	return nil
}
