package excelize

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSheet(t *testing.T) {
	f := NewFile()
	f.NewSheet("Sheet2")
	sheetID := f.NewSheet("sheet2")
	f.SetActiveSheet(sheetID)
	// delete original sheet
	f.DeleteSheet(f.GetSheetName(f.GetSheetIndex("Sheet1")))
	assert.NoError(t, f.SaveAs(filepath.Join("test", "TestNewSheet.xlsx")))
	// create new worksheet with already exists name
	assert.Equal(t, f.GetSheetIndex("Sheet2"), f.NewSheet("Sheet2"))
	// create new worksheet with empty sheet name
	assert.Equal(t, -1, f.NewSheet(":\\/?*[]"))
}

func TestSetPane(t *testing.T) {
	f := NewFile()
	assert.NoError(t, f.SetPanes("Sheet1", `{"freeze":false,"split":false}`))
	f.NewSheet("Panes 2")
	assert.NoError(t, f.SetPanes("Panes 2", `{"freeze":true,"split":false,"x_split":1,"y_split":0,"top_left_cell":"B1","active_pane":"topRight","panes":[{"sqref":"K16","active_cell":"K16","pane":"topRight"}]}`))
	f.NewSheet("Panes 3")
	assert.NoError(t, f.SetPanes("Panes 3", `{"freeze":false,"split":true,"x_split":3270,"y_split":1800,"top_left_cell":"N57","active_pane":"bottomLeft","panes":[{"sqref":"I36","active_cell":"I36"},{"sqref":"G33","active_cell":"G33","pane":"topRight"},{"sqref":"J60","active_cell":"J60","pane":"bottomLeft"},{"sqref":"O60","active_cell":"O60","pane":"bottomRight"}]}`))
	f.NewSheet("Panes 4")
	assert.NoError(t, f.SetPanes("Panes 4", `{"freeze":true,"split":false,"x_split":0,"y_split":9,"top_left_cell":"A34","active_pane":"bottomLeft","panes":[{"sqref":"A11:XFD11","active_cell":"A11","pane":"bottomLeft"}]}`))
	assert.EqualError(t, f.SetPanes("Panes 4", ""), "unexpected end of JSON input")
	assert.EqualError(t, f.SetPanes("SheetN", ""), "sheet SheetN does not exist")
	assert.NoError(t, f.SaveAs(filepath.Join("test", "TestSetPane.xlsx")))
	// Test add pane on empty sheet views worksheet
	f = NewFile()
	f.checked = nil
	f.Sheet.Delete("xl/worksheets/sheet1.xml")
	f.Pkg.Store("xl/worksheets/sheet1.xml", []byte(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData/></worksheet>`))
	assert.NoError(t, f.SetPanes("Sheet1", `{"freeze":true,"split":false,"x_split":1,"y_split":0,"top_left_cell":"B1","active_pane":"topRight","panes":[{"sqref":"K16","active_cell":"K16","pane":"topRight"}]}`))
}

func TestSearchSheet(t *testing.T) {
	f, err := OpenFile(filepath.Join("test", "SharedStrings.xlsx"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// Test search in a not exists worksheet.
	_, err = f.SearchSheet("Sheet4", "")
	assert.EqualError(t, err, "sheet Sheet4 does not exist")
	var expected []string
	// Test search a not exists value.
	result, err := f.SearchSheet("Sheet1", "X")
	assert.NoError(t, err)
	assert.EqualValues(t, expected, result)
	result, err = f.SearchSheet("Sheet1", "A")
	assert.NoError(t, err)
	assert.EqualValues(t, []string{"A1"}, result)
	// Test search the coordinates where the numerical value in the range of
	// "0-9" of Sheet1 is described by regular expression:
	result, err = f.SearchSheet("Sheet1", "[0-9]", true)
	assert.NoError(t, err)
	assert.EqualValues(t, expected, result)
	assert.NoError(t, f.Close())

	// Test search worksheet data after set cell value
	f = NewFile()
	assert.NoError(t, f.SetCellValue("Sheet1", "A1", true))
	_, err = f.SearchSheet("Sheet1", "")
	assert.NoError(t, err)

	f = NewFile()
	f.Sheet.Delete("xl/worksheets/sheet1.xml")
	f.Pkg.Store("xl/worksheets/sheet1.xml", []byte(`<worksheet><sheetData><row r="A"><c r="2" t="str"><v>A</v></c></row></sheetData></worksheet>`))
	f.checked = nil
	result, err = f.SearchSheet("Sheet1", "A")
	assert.EqualError(t, err, "strconv.Atoi: parsing \"A\": invalid syntax")
	assert.Equal(t, []string(nil), result)

	f.Pkg.Store("xl/worksheets/sheet1.xml", []byte(`<worksheet><sheetData><row r="2"><c r="A" t="str"><v>A</v></c></row></sheetData></worksheet>`))
	result, err = f.SearchSheet("Sheet1", "A")
	assert.EqualError(t, err, newCellNameToCoordinatesError("A", newInvalidCellNameError("A")).Error())
	assert.Equal(t, []string(nil), result)

	f.Pkg.Store("xl/worksheets/sheet1.xml", []byte(`<worksheet><sheetData><row r="0"><c r="A1" t="str"><v>A</v></c></row></sheetData></worksheet>`))
	result, err = f.SearchSheet("Sheet1", "A")
	assert.EqualError(t, err, "invalid cell reference [1, 0]")
	assert.Equal(t, []string(nil), result)
}

func TestSetPageLayout(t *testing.T) {
	f := NewFile()
	assert.NoError(t, f.SetPageLayout("Sheet1", nil))
	ws, ok := f.Sheet.Load("xl/worksheets/sheet1.xml")
	assert.True(t, ok)
	ws.(*xlsxWorksheet).PageSetUp = nil
	expected := PageLayoutOptions{
		Size:            intPtr(1),
		Orientation:     stringPtr("landscape"),
		FirstPageNumber: uintPtr(1),
		AdjustTo:        uintPtr(120),
		FitToHeight:     intPtr(2),
		FitToWidth:      intPtr(2),
		BlackAndWhite:   boolPtr(true),
	}
	assert.NoError(t, f.SetPageLayout("Sheet1", &expected))
	opts, err := f.GetPageLayout("Sheet1")
	assert.NoError(t, err)
	assert.Equal(t, expected, opts)
	// Test set page layout on not exists worksheet.
	assert.EqualError(t, f.SetPageLayout("SheetN", nil), "sheet SheetN does not exist")
}

func TestGetPageLayout(t *testing.T) {
	f := NewFile()
	// Test get page layout on not exists worksheet.
	_, err := f.GetPageLayout("SheetN")
	assert.EqualError(t, err, "sheet SheetN does not exist")
}

func TestSetHeaderFooter(t *testing.T) {
	f := NewFile()
	assert.NoError(t, f.SetCellStr("Sheet1", "A1", "Test SetHeaderFooter"))
	// Test set header and footer on not exists worksheet.
	assert.EqualError(t, f.SetHeaderFooter("SheetN", nil), "sheet SheetN does not exist")
	// Test set header and footer with illegal setting.
	assert.EqualError(t, f.SetHeaderFooter("Sheet1", &HeaderFooterOptions{
		OddHeader: strings.Repeat("c", MaxFieldLength+1),
	}), newFieldLengthError("OddHeader").Error())

	assert.NoError(t, f.SetHeaderFooter("Sheet1", nil))
	text := strings.Repeat("一", MaxFieldLength)
	assert.NoError(t, f.SetHeaderFooter("Sheet1", &HeaderFooterOptions{
		OddHeader:   text,
		OddFooter:   text,
		EvenHeader:  text,
		EvenFooter:  text,
		FirstHeader: text,
	}))
	assert.NoError(t, f.SetHeaderFooter("Sheet1", &HeaderFooterOptions{
		DifferentFirst:   true,
		DifferentOddEven: true,
		OddHeader:        "&R&P",
		OddFooter:        "&C&F",
		EvenHeader:       "&L&P",
		EvenFooter:       "&L&D&R&T",
		FirstHeader:      `&CCenter &"-,Bold"Bold&"-,Regular"HeaderU+000A&D`,
	}))
	assert.NoError(t, f.SaveAs(filepath.Join("test", "TestSetHeaderFooter.xlsx")))
}

func TestDefinedName(t *testing.T) {
	f := NewFile()
	assert.NoError(t, f.SetDefinedName(&DefinedName{
		Name:     "Amount",
		RefersTo: "Sheet1!$A$2:$D$5",
		Comment:  "defined name comment",
		Scope:    "Sheet1",
	}))
	assert.NoError(t, f.SetDefinedName(&DefinedName{
		Name:     "Amount",
		RefersTo: "Sheet1!$A$2:$D$5",
		Comment:  "defined name comment",
	}))
	assert.EqualError(t, f.SetDefinedName(&DefinedName{
		Name:     "Amount",
		RefersTo: "Sheet1!$A$2:$D$5",
		Comment:  "defined name comment",
	}), ErrDefinedNameDuplicate.Error())
	assert.EqualError(t, f.DeleteDefinedName(&DefinedName{
		Name: "No Exist Defined Name",
	}), ErrDefinedNameScope.Error())
	assert.Exactly(t, "Sheet1!$A$2:$D$5", f.GetDefinedName()[1].RefersTo)
	assert.NoError(t, f.DeleteDefinedName(&DefinedName{
		Name: "Amount",
	}))
	assert.Exactly(t, "Sheet1!$A$2:$D$5", f.GetDefinedName()[0].RefersTo)
	assert.Exactly(t, 1, len(f.GetDefinedName()))
	assert.NoError(t, f.SaveAs(filepath.Join("test", "TestDefinedName.xlsx")))
}

func TestGroupSheets(t *testing.T) {
	f := NewFile()
	sheets := []string{"Sheet2", "Sheet3"}
	for _, sheet := range sheets {
		f.NewSheet(sheet)
	}
	assert.EqualError(t, f.GroupSheets([]string{"Sheet1", "SheetN"}), "sheet SheetN does not exist")
	assert.EqualError(t, f.GroupSheets([]string{"Sheet2", "Sheet3"}), "group worksheet must contain an active worksheet")
	assert.NoError(t, f.GroupSheets([]string{"Sheet1", "Sheet2"}))
	assert.NoError(t, f.SaveAs(filepath.Join("test", "TestGroupSheets.xlsx")))
}

func TestUngroupSheets(t *testing.T) {
	f := NewFile()
	sheets := []string{"Sheet2", "Sheet3", "Sheet4", "Sheet5"}
	for _, sheet := range sheets {
		f.NewSheet(sheet)
	}
	assert.NoError(t, f.UngroupSheets())
}

func TestInsertPageBreak(t *testing.T) {
	f := NewFile()
	assert.NoError(t, f.InsertPageBreak("Sheet1", "A1"))
	assert.NoError(t, f.InsertPageBreak("Sheet1", "B2"))
	assert.NoError(t, f.InsertPageBreak("Sheet1", "C3"))
	assert.NoError(t, f.InsertPageBreak("Sheet1", "C3"))
	assert.EqualError(t, f.InsertPageBreak("Sheet1", "A"), newCellNameToCoordinatesError("A", newInvalidCellNameError("A")).Error())
	assert.EqualError(t, f.InsertPageBreak("SheetN", "C3"), "sheet SheetN does not exist")
	assert.NoError(t, f.SaveAs(filepath.Join("test", "TestInsertPageBreak.xlsx")))
}

func TestRemovePageBreak(t *testing.T) {
	f := NewFile()
	assert.NoError(t, f.RemovePageBreak("Sheet1", "A2"))

	assert.NoError(t, f.InsertPageBreak("Sheet1", "A2"))
	assert.NoError(t, f.InsertPageBreak("Sheet1", "B2"))
	assert.NoError(t, f.RemovePageBreak("Sheet1", "A1"))
	assert.NoError(t, f.RemovePageBreak("Sheet1", "B2"))

	assert.NoError(t, f.InsertPageBreak("Sheet1", "C3"))
	assert.NoError(t, f.RemovePageBreak("Sheet1", "C3"))

	assert.NoError(t, f.InsertPageBreak("Sheet1", "A3"))
	assert.NoError(t, f.RemovePageBreak("Sheet1", "B3"))
	assert.NoError(t, f.RemovePageBreak("Sheet1", "A3"))

	f.NewSheet("Sheet2")
	assert.NoError(t, f.InsertPageBreak("Sheet2", "B2"))
	assert.NoError(t, f.InsertPageBreak("Sheet2", "C2"))
	assert.NoError(t, f.RemovePageBreak("Sheet2", "B2"))

	assert.EqualError(t, f.RemovePageBreak("Sheet1", "A"), newCellNameToCoordinatesError("A", newInvalidCellNameError("A")).Error())
	assert.EqualError(t, f.RemovePageBreak("SheetN", "C3"), "sheet SheetN does not exist")
	assert.NoError(t, f.SaveAs(filepath.Join("test", "TestRemovePageBreak.xlsx")))
}

func TestGetSheetName(t *testing.T) {
	f, err := OpenFile(filepath.Join("test", "Book1.xlsx"))
	assert.NoError(t, err)
	assert.Equal(t, "Sheet1", f.GetSheetName(0))
	assert.Equal(t, "Sheet2", f.GetSheetName(1))
	assert.Equal(t, "", f.GetSheetName(-1))
	assert.Equal(t, "", f.GetSheetName(2))
	assert.NoError(t, f.Close())
}

func TestGetSheetMap(t *testing.T) {
	expectedMap := map[int]string{
		1: "Sheet1",
		2: "Sheet2",
	}
	f, err := OpenFile(filepath.Join("test", "Book1.xlsx"))
	assert.NoError(t, err)
	sheetMap := f.GetSheetMap()
	for idx, name := range sheetMap {
		assert.Equal(t, expectedMap[idx], name)
	}
	assert.Equal(t, len(sheetMap), 2)
	assert.NoError(t, f.Close())
}

func TestSetActiveSheet(t *testing.T) {
	f := NewFile()
	f.WorkBook.BookViews = nil
	f.SetActiveSheet(1)
	f.WorkBook.BookViews = &xlsxBookViews{WorkBookView: []xlsxWorkBookView{}}
	ws, ok := f.Sheet.Load("xl/worksheets/sheet1.xml")
	assert.True(t, ok)
	ws.(*xlsxWorksheet).SheetViews = &xlsxSheetViews{SheetView: []xlsxSheetView{}}
	f.SetActiveSheet(1)
	ws, ok = f.Sheet.Load("xl/worksheets/sheet1.xml")
	assert.True(t, ok)
	ws.(*xlsxWorksheet).SheetViews = nil
	f.SetActiveSheet(1)
	f = NewFile()
	f.SetActiveSheet(-1)
	assert.Equal(t, f.GetActiveSheetIndex(), 0)

	f = NewFile()
	f.WorkBook.BookViews = nil
	idx := f.NewSheet("Sheet2")
	ws, ok = f.Sheet.Load("xl/worksheets/sheet2.xml")
	assert.True(t, ok)
	ws.(*xlsxWorksheet).SheetViews = &xlsxSheetViews{SheetView: []xlsxSheetView{}}
	f.SetActiveSheet(idx)
}

func TestSetSheetName(t *testing.T) {
	f := NewFile()
	// Test set worksheet with the same name.
	f.SetSheetName("Sheet1", "Sheet1")
	assert.Equal(t, "Sheet1", f.GetSheetName(0))
}

func TestWorksheetWriter(t *testing.T) {
	f := NewFile()
	// Test set cell value with alternate content
	f.Sheet.Delete("xl/worksheets/sheet1.xml")
	worksheet := xml.Header + `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheetData><row r="1"><c r="A1"><v>%d</v></c></row></sheetData><mc:AlternateContent xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006"><mc:Choice xmlns:a14="http://schemas.microsoft.com/office/drawing/2010/main" Requires="a14"><xdr:twoCellAnchor editAs="oneCell"></xdr:twoCellAnchor></mc:Choice><mc:Fallback/></mc:AlternateContent></worksheet>`
	f.Pkg.Store("xl/worksheets/sheet1.xml", []byte(fmt.Sprintf(worksheet, 1)))
	f.checked = nil
	assert.NoError(t, f.SetCellValue("Sheet1", "A1", 2))
	f.workSheetWriter()
	value, ok := f.Pkg.Load("xl/worksheets/sheet1.xml")
	assert.True(t, ok)
	assert.Equal(t, fmt.Sprintf(worksheet, 2), string(value.([]byte)))
}

func TestGetWorkbookPath(t *testing.T) {
	f := NewFile()
	f.Pkg.Delete("_rels/.rels")
	assert.Equal(t, "", f.getWorkbookPath())
}

func TestGetWorkbookRelsPath(t *testing.T) {
	f := NewFile()
	f.Pkg.Delete("xl/_rels/.rels")
	f.Pkg.Store("_rels/.rels", []byte(xml.Header+`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://purl.oclc.org/ooxml/officeDocument/relationships/officeDocument" Target="/workbook.xml"/></Relationships>`))
	assert.Equal(t, "_rels/workbook.xml.rels", f.getWorkbookRelsPath())
}

func TestDeleteSheet(t *testing.T) {
	f := NewFile()
	f.SetActiveSheet(f.NewSheet("Sheet2"))
	f.NewSheet("Sheet3")
	f.DeleteSheet("Sheet1")
	assert.Equal(t, "Sheet2", f.GetSheetName(f.GetActiveSheetIndex()))
	assert.NoError(t, f.SaveAs(filepath.Join("test", "TestDeleteSheet.xlsx")))
	// Test with auto filter defined names
	f = NewFile()
	f.NewSheet("Sheet2")
	f.NewSheet("Sheet3")
	assert.NoError(t, f.SetCellValue("Sheet1", "A1", "A"))
	assert.NoError(t, f.SetCellValue("Sheet2", "A1", "A"))
	assert.NoError(t, f.SetCellValue("Sheet3", "A1", "A"))
	assert.NoError(t, f.AutoFilter("Sheet1", "A1", "A1", ""))
	assert.NoError(t, f.AutoFilter("Sheet2", "A1", "A1", ""))
	assert.NoError(t, f.AutoFilter("Sheet3", "A1", "A1", ""))
	f.DeleteSheet("Sheet2")
	f.DeleteSheet("Sheet1")
	assert.NoError(t, f.SaveAs(filepath.Join("test", "TestDeleteSheet2.xlsx")))
}

func TestDeleteAndAdjustDefinedNames(t *testing.T) {
	deleteAndAdjustDefinedNames(nil, 0)
	deleteAndAdjustDefinedNames(&xlsxWorkbook{}, 0)
}

func TestGetSheetID(t *testing.T) {
	file := NewFile()
	file.NewSheet("Sheet1")
	id := file.getSheetID("sheet1")
	assert.NotEqual(t, -1, id)
}

func BenchmarkNewSheet(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			newSheetWithSet()
		}
	})
}

func newSheetWithSet() {
	file := NewFile()
	file.NewSheet("sheet1")
	for i := 0; i < 1000; i++ {
		_ = file.SetCellInt("sheet1", "A"+strconv.Itoa(i+1), i)
	}
	file = nil
}

func BenchmarkFile_SaveAs(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			newSheetWithSave()
		}
	})
}

func newSheetWithSave() {
	file := NewFile()
	file.NewSheet("sheet1")
	for i := 0; i < 1000; i++ {
		_ = file.SetCellInt("sheet1", "A"+strconv.Itoa(i+1), i)
	}
	_ = file.Save()
}

func TestAttrValToBool(t *testing.T) {
	_, err := attrValToBool("hidden", []xml.Attr{
		{Name: xml.Name{Local: "hidden"}},
	})
	assert.EqualError(t, err, `strconv.ParseBool: parsing "": invalid syntax`)

	got, err := attrValToBool("hidden", []xml.Attr{
		{Name: xml.Name{Local: "hidden"}, Value: "1"},
	})
	assert.NoError(t, err)
	assert.Equal(t, true, got)
}

func TestAttrValToFloat(t *testing.T) {
	_, err := attrValToFloat("ht", []xml.Attr{
		{Name: xml.Name{Local: "ht"}},
	})
	assert.EqualError(t, err, `strconv.ParseFloat: parsing "": invalid syntax`)

	got, err := attrValToFloat("ht", []xml.Attr{
		{Name: xml.Name{Local: "ht"}, Value: "42.1"},
	})
	assert.NoError(t, err)
	assert.Equal(t, 42.1, got)
}
