package excel

import (
	"bytes"
	"fmt"
	"math"
	"sort"
	"strings"
	"telegram_bot/internal/modules/report/worker"
	"time"

	"telegram_bot/internal/modules/report"

	"github.com/xuri/excelize/v2"
)

type Generator struct{}

type StrictSubmissionsStream struct {
	f                *excelize.File
	sheet            string
	currentRow       int
	currentUser      string
	userSubCount     int
	taskCounts       map[string]int
	mainTitleStyle   int
	sectionStyle     int
	userHeaderStyle  int
	tableHeaderStyle int
}

func New() *Generator {
	return &Generator{}
}

func (g *Generator) GenerateStatsReport(data []report.UserStat) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Статистика за неделю"
	f.SetSheetName("Sheet1", sheet)

	headers := []interface{}{"ID", "Имя", "Роль", "Сдано ДЗ (за 7 дней)", "Конспектов (за 7 дней)", "Всего файлов (за 7 дней)", "Дата регистрации"}
	f.SetSheetRow(sheet, "A1", &headers)

	for i, u := range data {
		row := i + 2
		rowData := []interface{}{
			u.UserID, u.Name, u.Role, u.HomeworkCount, u.NotesCount, u.FilesCount, u.RegisteredAt.Format(time.DateTime),
		}
		f.SetSheetRow(sheet, fmt.Sprintf("A%d", row), &rowData)
	}

	f.SetColWidth(sheet, "A", "G", 20)

	if len(data) > 0 {
		f.AddTable(sheet, &excelize.Table{
			Range:     fmt.Sprintf("A1:G%d", len(data)+1),
			Name:      "UsersTable",
			StyleName: "TableStyleMedium2",
		})
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g *Generator) NewStrictSubmissionsStream() (worker.StrictReportStream, error) {
	f := excelize.NewFile()
	sheet := "История сдач"
	f.SetSheetName("Sheet1", sheet)

	mainTitleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 14},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1F497D"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	sectionStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#366092"}, Pattern: 1},
	})
	userHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#DCE6F1"}, Pattern: 1},
	})
	tableHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#F2F2F2"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
		},
	})

	return &StrictSubmissionsStream{
		f:                f,
		sheet:            sheet,
		currentRow:       20,
		taskCounts:       make(map[string]int),
		mainTitleStyle:   mainTitleStyle,
		sectionStyle:     sectionStyle,
		userHeaderStyle:  userHeaderStyle,
		tableHeaderStyle: tableHeaderStyle,
	}, nil
}

func (s *StrictSubmissionsStream) WriteRow(username, role string, detail report.SubmissionDetail) error {
	if s.currentUser != username {
		s.currentUser = username
		s.userSubCount = 0

		if s.currentRow > 20 {
			s.currentRow++
		}

		s.f.SetCellValue(s.sheet, fmt.Sprintf("A%d", s.currentRow), fmt.Sprintf("Ученик: %s", username))
		s.f.SetCellStyle(s.sheet, fmt.Sprintf("A%d", s.currentRow), fmt.Sprintf("H%d", s.currentRow), s.userHeaderStyle)
		s.currentRow++

		headers := []interface{}{"№", "Дата и время", "Тип работы", "Задание", "Файлов", "Комментарий", "Статус", "ID сдачи"}
		s.f.SetSheetRow(s.sheet, fmt.Sprintf("A%d", s.currentRow), &headers)
		s.f.SetRowStyle(s.sheet, s.currentRow, s.currentRow, s.tableHeaderStyle)
		s.currentRow++
	}

	s.userSubCount++

	nameLower := strings.ToLower(detail.TaskName)
	if !strings.Contains(nameLower, "созвон") {
		s.taskCounts[detail.TaskName]++
	}

	cleanType := strings.ReplaceAll(strings.ReplaceAll(detail.Type, "📚 ", ""), "📝 ", "")
	cleanStatus := strings.ReplaceAll(strings.ReplaceAll(detail.Status, "⏳ ", ""), "✅ ", "")

	rowData := []interface{}{
		s.userSubCount, detail.DateTime, cleanType, detail.TaskName, detail.FilesCount, detail.Comment, cleanStatus, detail.SubmissionID,
	}
	s.f.SetSheetRow(s.sheet, fmt.Sprintf("A%d", s.currentRow), &rowData)
	s.currentRow++

	return nil
}

func (s *StrictSubmissionsStream) Finish(data *report.StrictReportData) ([]byte, error) {
	defer s.f.Close()

	row := 1
	s.f.SetCellValue(s.sheet, fmt.Sprintf("A%d", row), data.ReportTitle)
	s.f.MergeCell(s.sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("H%d", row))
	s.f.SetCellStyle(s.sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), s.mainTitleStyle)
	row++

	meta := [][]interface{}{
		{"Дата формирования:", data.GenerationDate},
		{"Курс:", data.CourseName},
		{"Куратор:", data.CuratorName},
		{"Период:", data.Period},
	}
	for _, m := range meta {
		s.f.SetSheetRow(s.sheet, fmt.Sprintf("A%d", row), &m)
		row++
	}
	row++

	s.f.SetCellValue(s.sheet, fmt.Sprintf("A%d", row), "ОБЩАЯ СТАТИСТИКА ЗА ПЕРИОД")
	s.f.SetCellStyle(s.sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("H%d", row), s.sectionStyle)
	row++

	stats := [][]interface{}{
		{"Всего пользователей:", data.TotalUsers, "", "Всего сдач за период:", data.TotalSubmissions},
		{"Студентов на курсе:", data.CourseStudents, "", "Сдач ДЗ:", data.HWSubmissions},
		{"Активных студентов:", data.ActiveStudents, "", "Сдач конспектов:", data.NotesSubmissions},
		{"Среднее сдач на юзера:", fmt.Sprintf("%.1f", data.AvgSubmissions), "", "Распределение (ДЗ/Консп):", data.HWNotesRatio},
	}
	for _, st := range stats {
		s.f.SetSheetRow(s.sheet, fmt.Sprintf("A%d", row), &st)
		row++
	}

	s.f.SetCellValue(s.sheet, "A18", "ДЕТАЛИЗАЦИЯ ПО УЧЕНИКАМ")
	s.f.SetCellStyle(s.sheet, "A18", "H18", s.sectionStyle)

	s.f.SetColWidth(s.sheet, "A", "B", 20)
	s.f.SetColWidth(s.sheet, "C", "C", 12)
	s.f.SetColWidth(s.sheet, "D", "D", 35)
	s.f.SetColWidth(s.sheet, "E", "E", 10)
	s.f.SetColWidth(s.sheet, "F", "F", 25)
	s.f.SetColWidth(s.sheet, "G", "H", 15)

	sheetAnalytics := "Аналитика"
	s.f.NewSheet(sheetAnalytics)

	redAlertStyle, _ := s.f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFC7CE"}, Pattern: 1},
		Font: &excelize.Font{Color: "#9C0006", Bold: true},
	})
	goodStyle, _ := s.f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#006100"},
	})

	s.f.SetSheetRow(sheetAnalytics, "A1", &[]interface{}{"Задание (без созвонов)", "Количество сдач", "Анализ сложности"})
	s.f.SetRowStyle(sheetAnalytics, 1, 1, s.tableHeaderStyle)

	type taskStat struct {
		Name  string
		Count int
	}
	var tasks []taskStat
	sum := 0
	for k, v := range s.taskCounts {
		tasks = append(tasks, taskStat{Name: k, Count: v})
		sum += v
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Count > tasks[j].Count })

	avg := 0.0
	if len(tasks) > 0 {
		avg = float64(sum) / float64(len(tasks))
	}
	lowThreshold := math.Ceil(avg * 0.5)

	for i, t := range tasks {
		r := i + 2
		analysis := "В норме"

		s.f.SetCellValue(sheetAnalytics, fmt.Sprintf("A%d", r), t.Name)
		s.f.SetCellValue(sheetAnalytics, fmt.Sprintf("B%d", r), t.Count)

		if float64(t.Count) <= lowThreshold {
			analysis = "СЛОЖНОЕ (мало сдач!)"
			s.f.SetCellValue(sheetAnalytics, fmt.Sprintf("C%d", r), analysis)
			s.f.SetCellStyle(sheetAnalytics, fmt.Sprintf("A%d", r), fmt.Sprintf("C%d", r), redAlertStyle)
		} else {
			s.f.SetCellValue(sheetAnalytics, fmt.Sprintf("C%d", r), analysis)
			s.f.SetCellStyle(sheetAnalytics, fmt.Sprintf("C%d", r), fmt.Sprintf("C%d", r), goodStyle)
		}
	}

	s.f.SetColWidth(sheetAnalytics, "A", "A", 40)
	s.f.SetColWidth(sheetAnalytics, "B", "C", 20)

	if len(tasks) > 0 {
		endRow := len(tasks) + 1
		varyColors := true

		err := s.f.AddChart(sheetAnalytics, "E2", &excelize.Chart{
			Type: excelize.Col,
			Series: []excelize.ChartSeries{
				{
					Name:       "Аналитика!$B$1",
					Categories: fmt.Sprintf("Аналитика!$A$2:$A$%d", endRow),
					Values:     fmt.Sprintf("Аналитика!$B$2:$B$%d", endRow),
				},
			},
			Format: excelize.GraphicOptions{
				ScaleX: 1.8,
				ScaleY: 1.4,
			},
			Legend: excelize.ChartLegend{
				Position: "none",
			},
			Title: []excelize.RichTextRun{
				{
					Text: "📊 Популярность заданий",
				},
			},
			VaryColors: &varyColors,
		})
		if err != nil {
			fmt.Println("Ошибка построения графика:", err)
		}
	}

	s.f.SetActiveSheet(0)
	var buf bytes.Buffer
	if err := s.f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g *Generator) GenerateWeeklyReportsExcel(data []report.WeeklyReportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Рефлексии"
	f.SetSheetName("Sheet1", sheet)

	headers := []interface{}{"Дата", "Ученик", "Куратор", "Текст отчета"}
	f.SetSheetRow(sheet, "A1", &headers)

	wrapStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"},
	})

	for i, rep := range data {
		row := i + 2
		rowData := []interface{}{rep.Date, rep.StudentName, rep.CuratorName, rep.Text}
		f.SetSheetRow(sheet, fmt.Sprintf("A%d", row), &rowData)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), wrapStyle)
	}

	f.SetColWidth(sheet, "A", "A", 18)
	f.SetColWidth(sheet, "B", "C", 20)
	f.SetColWidth(sheet, "D", "D", 80)

	if len(data) > 0 {
		f.AddTable(sheet, &excelize.Table{
			Range:     fmt.Sprintf("A1:D%d", len(data)+1),
			Name:      "WeeklyReportsTable",
			StyleName: "TableStyleMedium6",
		})
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g *Generator) GenerateStudentSheetsExcel(data []report.StudentSheetRecord) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	curatorMap := make(map[string][]report.StudentSheetRecord)
	for _, rec := range data {
		curatorMap[rec.CuratorName] = append(curatorMap[rec.CuratorName], rec)
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#4F81BD"}, Pattern: 1},
	})

	for curatorName, students := range curatorMap {
		sheetName := curatorName
		if len(sheetName) > 30 {
			sheetName = sheetName[:30]
		}

		sheetName = strings.ReplaceAll(sheetName, ":", "")
		sheetName = strings.ReplaceAll(sheetName, "/", "")

		f.NewSheet(sheetName)

		headers := []interface{}{"Ученик", "Username", "Курс", "Сдано ДЗ", "Сдано Конспектов", "Дата регистрации"}
		f.SetSheetRow(sheetName, "A1", &headers)
		f.SetRowStyle(sheetName, 1, 1, headerStyle)

		for i, student := range students {
			row := i + 2
			rowData := []interface{}{
				strings.TrimSpace(student.StudentName),
				"@" + student.Username,
				student.CourseName,
				student.HWCount,
				student.NotesCount,
				student.RegisteredAt,
			}
			f.SetSheetRow(sheetName, fmt.Sprintf("A%d", row), &rowData)
		}

		f.SetColWidth(sheetName, "A", "A", 25)
		f.SetColWidth(sheetName, "B", "C", 20)
		f.SetColWidth(sheetName, "D", "E", 15)
		f.SetColWidth(sheetName, "F", "F", 15)

		if len(students) > 0 {
			f.AddTable(sheetName, &excelize.Table{
				Range:     fmt.Sprintf("A1:F%d", len(students)+1),
				Name:      "Table_" + strings.ReplaceAll(sheetName, " ", ""),
				StyleName: "TableStyleLight9",
			})
		}
	}

	f.DeleteSheet("Sheet1")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
