package excel

import (
	"fmt"
	"time"

	"telegram_bot/internal/modules/report"

	"github.com/xuri/excelize/v2"
)

type Generator struct{}

func New() *Generator {
	return &Generator{}
}

func (g *Generator) GenerateStatsReport(data []report.UserStat) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Статистика"
	index, _ := f.NewSheet(sheet)
	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1")

	headers := []string{"ID", "Имя", "Роль", "Сдано ДЗ", "Всего файлов", "Дата регистрации"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
	})
	f.SetRowStyle(sheet, 1, 1, style)

	for i, u := range data {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), u.UserID)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), u.Name)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), u.Role)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), u.HomeworkCount)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), u.FilesCount)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), u.RegisteredAt.Format(time.DateTime))
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
