package word

import (
	"bytes"
	"fmt"
	"time"

	"github.com/unidoc/unioffice/color"
	"github.com/unidoc/unioffice/document"
	"github.com/unidoc/unioffice/measurement"
	"github.com/unidoc/unioffice/schema/soo/wml"
)

type Generator struct{}

func New() *Generator {
	return &Generator{}
}

func (g *Generator) GenerateWeeklyReport(studentName string, reportText string) ([]byte, error) {
	doc := document.New()
	defer doc.Close()

	pTitle := doc.AddParagraph()
	pTitle.Properties().SetAlignment(wml.ST_JcCenter)

	rTitle := pTitle.AddRun()
	rTitle.AddText("Еженедельный отчет")
	rTitle.Properties().SetBold(true)
	rTitle.Properties().SetSize(16 * measurement.Point)

	doc.AddParagraph()

	pStudent := doc.AddParagraph()

	rLabel := pStudent.AddRun()
	rLabel.AddText("Студент: ")
	rLabel.Properties().SetBold(true)

	rName := pStudent.AddRun()
	rName.AddText(studentName)

	pDate := doc.AddParagraph()

	rDateLabel := pDate.AddRun()
	rDateLabel.AddText("Дата формирования: ")
	rDateLabel.Properties().SetBold(true)

	rDateValue := pDate.AddRun()
	rDateValue.AddText(time.Now().Format("02.01.2006"))

	doc.AddParagraph()

	pHeader := doc.AddParagraph()
	rHeader := pHeader.AddRun()
	rHeader.AddText("Содержание отчета:")
	rHeader.Properties().SetBold(true)
	rHeader.Properties().SetSize(12 * measurement.Point)

	rHeader.Properties().SetColor(color.RGB(128, 128, 128))

	pContent := doc.AddParagraph()
	rContent := pContent.AddRun()
	rContent.AddText(reportText)

	doc.AddParagraph()

	pFooter := doc.AddParagraph()
	pFooter.Properties().SetAlignment(wml.ST_JcRight)

	rFooter := pFooter.AddRun()
	rFooter.AddText("Сгенерировано ботом")
	rFooter.Properties().SetItalic(true)

	rFooter.Properties().SetColor(color.RGB(169, 169, 169))
	rFooter.Properties().SetSize(8 * measurement.Point)

	var buf bytes.Buffer
	if err := doc.Save(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate docx: %w", err)
	}

	return buf.Bytes(), nil
}
