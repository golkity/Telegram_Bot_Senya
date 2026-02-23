package excel

import (
	"bytes"
	"fmt"
	"math"
	"sort"
	"strings"
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
	f.SetSheetName("Sheet1", sheet)

	headers := []interface{}{"ID", "Имя", "Роль", "Сдано ДЗ", "Всего файлов", "Дата регистрации"}
	f.SetSheetRow(sheet, "A1", &headers)

	for i, u := range data {
		row := i + 2
		rowData := []interface{}{
			u.UserID, u.Name, u.Role, u.HomeworkCount, u.FilesCount, u.RegisteredAt.Format(time.DateTime),
		}
		f.SetSheetRow(sheet, fmt.Sprintf("A%d", row), &rowData)
	}

	f.SetColWidth(sheet, "A", "F", 20)

	if len(data) > 0 {
		f.AddTable(sheet, &excelize.Table{
			Range:     fmt.Sprintf("A1:F%d", len(data)+1),
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

func (g *Generator) GenerateStrictSubmissionsReport(data *report.StrictReportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetHistory := "История сдач"
	f.SetSheetName("Sheet1", sheetHistory)

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

	row := 1

	f.SetCellValue(sheetHistory, fmt.Sprintf("A%d", row), data.ReportTitle)
	f.MergeCell(sheetHistory, fmt.Sprintf("A%d", row), fmt.Sprintf("H%d", row))
	f.SetCellStyle(sheetHistory, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), mainTitleStyle)
	row++

	meta := [][]interface{}{
		{"Дата формирования:", data.GenerationDate},
		{"Курс:", data.CourseName},
		{"Куратор:", data.CuratorName},
		{"Период:", data.Period},
	}
	for _, m := range meta {
		f.SetSheetRow(sheetHistory, fmt.Sprintf("A%d", row), &m)
		row++
	}
	row++

	f.SetCellValue(sheetHistory, fmt.Sprintf("A%d", row), "ОБЩАЯ СТАТИСТИКА ЗА ПЕРИОД")
	f.SetCellStyle(sheetHistory, fmt.Sprintf("A%d", row), fmt.Sprintf("H%d", row), sectionStyle)
	row++

	stats := [][]interface{}{
		{"Всего пользователей:", data.TotalUsers, "", "Всего сдач за период:", data.TotalSubmissions},
		{"Студентов на курсе:", data.CourseStudents, "", "Сдач ДЗ:", data.HWSubmissions},
		{"Активных студентов:", data.ActiveStudents, "", "Сдач конспектов:", data.NotesSubmissions},
		{"Среднее сдач на юзера:", fmt.Sprintf("%.1f", data.AvgSubmissions), "", "Распределение (ДЗ/Консп):", data.HWNotesRatio},
	}
	for _, s := range stats {
		f.SetSheetRow(sheetHistory, fmt.Sprintf("A%d", row), &s)
		row++
	}
	row += 2

	f.SetCellValue(sheetHistory, fmt.Sprintf("A%d", row), "ДЕТАЛИЗАЦИЯ ПО УЧЕНИКАМ")
	f.SetCellStyle(sheetHistory, fmt.Sprintf("A%d", row), fmt.Sprintf("H%d", row), sectionStyle)
	row++

	taskCounts := make(map[string]int)

	for _, user := range data.Users {
		f.SetCellValue(sheetHistory, fmt.Sprintf("A%d", row), fmt.Sprintf("%s (Сдач: %d)", user.Username, user.TotalCount))
		f.SetCellStyle(sheetHistory, fmt.Sprintf("A%d", row), fmt.Sprintf("H%d", row), userHeaderStyle)
		row++

		headers := []interface{}{"№", "Дата и время", "Тип работы", "Задание", "Файлов", "Комментарий", "Статус", "ID сдачи"}
		f.SetSheetRow(sheetHistory, fmt.Sprintf("A%d", row), &headers)
		f.SetRowStyle(sheetHistory, row, row, tableHeaderStyle)
		row++

		for _, sub := range user.Submissions {
			nameLower := strings.ToLower(sub.TaskName)
			if !strings.Contains(nameLower, "созвон") {
				taskCounts[sub.TaskName]++
			}

			cleanType := strings.ReplaceAll(strings.ReplaceAll(sub.Type, "📚 ", ""), "📝 ", "")
			cleanStatus := strings.ReplaceAll(strings.ReplaceAll(sub.Status, "⏳ ", ""), "✅ ", "")

			rowData := []interface{}{
				sub.Number, sub.DateTime, cleanType, sub.TaskName, sub.FilesCount, sub.Comment, cleanStatus, sub.SubmissionID,
			}
			f.SetSheetRow(sheetHistory, fmt.Sprintf("A%d", row), &rowData)
			row++
		}
		row++
	}

	f.SetColWidth(sheetHistory, "A", "B", 20)
	f.SetColWidth(sheetHistory, "C", "C", 12)
	f.SetColWidth(sheetHistory, "D", "D", 35)
	f.SetColWidth(sheetHistory, "E", "E", 10)
	f.SetColWidth(sheetHistory, "F", "F", 25)
	f.SetColWidth(sheetHistory, "G", "H", 15)

	sheetAnalytics := "Аналитика"
	f.NewSheet(sheetAnalytics)

	redAlertStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFC7CE"}, Pattern: 1},
		Font: &excelize.Font{Color: "#9C0006", Bold: true},
	})
	goodStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#006100"},
	})

	f.SetSheetRow(sheetAnalytics, "A1", &[]interface{}{"Задание (без созвонов)", "Количество сдач", "Анализ сложности"})
	f.SetRowStyle(sheetAnalytics, 1, 1, tableHeaderStyle)

	type taskStat struct {
		Name  string
		Count int
	}
	var tasks []taskStat
	sum := 0
	for k, v := range taskCounts {
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

		f.SetCellValue(sheetAnalytics, fmt.Sprintf("A%d", r), t.Name)
		f.SetCellValue(sheetAnalytics, fmt.Sprintf("B%d", r), t.Count)

		if float64(t.Count) <= lowThreshold {
			analysis = "СЛОЖНОЕ (мало сдач!)"
			f.SetCellValue(sheetAnalytics, fmt.Sprintf("C%d", r), analysis)
			f.SetCellStyle(sheetAnalytics, fmt.Sprintf("A%d", r), fmt.Sprintf("C%d", r), redAlertStyle)
		} else {
			f.SetCellValue(sheetAnalytics, fmt.Sprintf("C%d", r), analysis)
			f.SetCellStyle(sheetAnalytics, fmt.Sprintf("C%d", r), fmt.Sprintf("C%d", r), goodStyle)
		}
	}

	f.SetColWidth(sheetAnalytics, "A", "A", 40)
	f.SetColWidth(sheetAnalytics, "B", "C", 20)

	if len(tasks) > 0 {
		endRow := len(tasks) + 1

		varyColors := true

		err := f.AddChart(sheetAnalytics, "E2", &excelize.Chart{
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

	f.SetActiveSheet(0)
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
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
