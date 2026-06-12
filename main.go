package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

var translationDirection = 1 // 1: Lotin -> Kirill, 2: Kirill -> Lotin

func transliterateLatToCyr(text string) string {
	mapping := map[string]string{
		"sh": "ш", "ch": "ч", "ya": "я", "yo": "ё", "yu": "ю", "ts": "ц",
		"Sh": "Ш", "Ch": "Ч", "Ya": "Я", "Yo": "Ё", "Yu": "Ю", "Ts": "Ц",
		"SH": "Ш", "CH": "Ч", "YA": "Я", "YO": "Ё", "YU": "Ю", "TS": "Ц",
		"O'": "Ў", "O‘": "Ў", "o'": "ў", "o‘": "ў",
		"G'": "Ғ", "G‘": "Ғ", "g'": "ғ", "g‘": "ғ",
		"a": "а", "b": "б", "d": "д", "e": "е", "f": "ф", "g": "г",
		"h": "ҳ", "i": "и", "j": "ж", "k": "к", "l": "л", "m": "м",
		"n": "н", "o": "о", "p": "п", "q": "қ", "r": "р", "s": "с",
		"t": "т", "u": "у", "v": "в", "x": "х", "y": "й", "z": "з",
		"A": "А", "B": "Б", "D": "Д", "E": "Е", "F": "Ф", "G": "Г",
		"H": "Ҳ", "I": "И", "J": "Ж", "K": "К", "L": "Л", "M": "М",
		"N": "Н", "O": "О", "P": "П", "Q": "Қ", "R": "Р", "S": "С",
		"T": "Т", "U": "У", "V": "В", "X": "Х", "Y": "Й", "Z": "З",
		"'": "ъ", "’": "ъ",
	}

	for _, pair := range []string{"sh", "ch", "ya", "yo", "yu", "ts", "Sh", "Ch", "Ya", "Yo", "Yu", "Ts", "SH", "CH", "YA", "YO", "YU", "TS", "O'", "O‘", "o'", "o‘", "G'", "G‘", "g'", "g‘"} {
		text = strings.ReplaceAll(text, pair, mapping[pair])
	}
	for lat, cyr := range mapping {
		if len(lat) == 1 {
			text = strings.ReplaceAll(text, lat, cyr)
		}
	}
	return text
}

func transliterateCyrToLat(text string) string {
	mapping := map[string]string{
		"ш": "sh", "ч": "ch", "я": "ya", "ё": "yo", "ю": "yu", "ц": "ts",
		"Ш": "Sh", "Ч": "Ch", "Я": "Ya", "Ё": "Yo", "Ю": "Yu", "Ц": "Ts",
		"Ў": "O'", "ў": "o'", "Ғ": "G'", "ғ": "g'", "Ъ": "'", "ъ": "'", "Ь": "", "ь": "", "Э": "E", "э": "e",
		"а": "a", "б": "b", "д": "d", "е": "e", "ф": "f", "г": "g",
		"ҳ": "h", "и": "i", "ж": "j", "к": "k", "л": "l", "м": "m",
		"н": "n", "о": "o", "п": "p", "қ": "q", "р": "r", "с": "s",
		"т": "t", "у": "u", "в": "v", "х": "x", "й": "y", "з": "z",
		"А": "A", "Б": "B", "Д": "D", "Е": "E", "Ф": "F", "Г": "G",
		"Ҳ": "H", "И": "I", "Ж": "J", "К": "K", "Л": "L", "М": "M",
		"Н": "N", "О": "O", "П": "P", "Қ": "Q", "Р": "R", "С": "S",
		"Т": "T", "У": "U", "В": "V", "Х": "X", "Й": "Y", "З": "Z",
	}

	for cyr, lat := range mapping {
		text = strings.ReplaceAll(text, cyr, lat)
	}
	return text
}

func transliterate(text string) string {
	if translationDirection == 2 {
		return transliterateCyrToLat(text)
	}
	return transliterateLatToCyr(text)
}

func processXLSX(inputFile, outputFile string) error {
	f, err := excelize.OpenFile(inputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, sheetName := range f.GetSheetMap() {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			continue
		}
		for rowIndex, row := range rows {
			for colIndex, cellValue := range row {
				if cellValue != "" {
					newText := transliterate(cellValue)
					colName, err := excelize.ColumnNumberToName(colIndex + 1)
					if err != nil {
						continue
					}
					cellName := fmt.Sprintf("%s%d", colName, rowIndex+1)
					f.SetCellValue(sheetName, cellName, newText)
				}
			}
		}
	}
	return f.SaveAs(outputFile)
}

func replaceTextInXML(content []byte) []byte {
	xmlText := string(content)

	// DOCX tags
	xmlText = replaceInsideTags(xmlText, "<w:t>", "</w:t>")
	xmlText = replaceInsideTags(xmlText, "<w:t xml:space=\"preserve\">", "</w:t>")
	// PPTX tags
	xmlText = replaceInsideTags(xmlText, "<a:t>", "</a:t>")

	return []byte(xmlText)
}

func replaceInsideTags(xmlText, openTag, closeTag string) string {
	parts := strings.Split(xmlText, openTag)
	for i := 1; i < len(parts); i++ {
		subParts := strings.Split(parts[i], closeTag)
		if len(subParts) >= 2 {
			translated := transliterate(subParts[0])
			parts[i] = translated + closeTag + strings.Join(subParts[1:], closeTag)
		}
	}
	return strings.Join(parts, openTag)
}

func processZipXML(inputFile, outputFile string) error {
	r, err := zip.OpenReader(inputFile)
	if err != nil {
		return err
	}
	defer r.Close()

	newZipFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	w := zip.NewWriter(newZipFile)
	defer w.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}

		if strings.HasSuffix(f.Name, ".xml") && (strings.Contains(f.Name, "word/document.xml") ||
			strings.Contains(f.Name, "word/header") || strings.Contains(f.Name, "word/footer") ||
			strings.Contains(f.Name, "ppt/slides/slide")) {
			content = replaceTextInXML(content)
		}

		newFile, err := w.Create(f.Name)
		if err != nil {
			return err
		}
		_, err = newFile.Write(content)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Foydalanish: tarjimon.exe <hujjat_fayl_nomi>")
		fmt.Println("Yoki shunchaki Word/Excel faylini ustiga olib kelib tashlang (Drag and Drop)")
		fmt.Scanln()
		return
	}

	inputFile := os.Args[1]

	fmt.Println("-------------------------------------------")
	fmt.Println("Qaysi tomonga tarjima qilmoqchisiz?")
	fmt.Println("1) Lotin yozuvidan -> Kirill yozuviga")
	fmt.Println("2) Kirill yozuvidan -> Lotin yozuviga")
	fmt.Print("\nTanlovingiz (1 yoki 2) kiriting: ")

	var userChoice string
	fmt.Scanln(&userChoice)

	if userChoice == "2" {
		translationDirection = 2
	} else {
		translationDirection = 1
	}

	ext := strings.ToLower(filepath.Ext(inputFile))
	base := strings.TrimSuffix(inputFile, ext)

	var outputFile string
	if translationDirection == 2 {
		outputFile = base + "_lotin" + ext
	} else {
		outputFile = base + "_kirill" + ext
	}

	fmt.Printf("\nTarjima qilinmoqda: %s\n", filepath.Base(inputFile))

	var err error
	if ext == ".xlsx" {
		err = processXLSX(inputFile, outputFile)
	} else if ext == ".docx" || ext == ".pptx" {
		err = processZipXML(inputFile, outputFile)
	} else {
		fmt.Println("Xato: Faqat .docx, .xlsx va .pptx fayllarni qo'llab-quvvatlaydi!")
		fmt.Scanln()
		return
	}

	if err != nil {
		fmt.Printf("Xatolik yuz berdi: %v\n", err)
	} else {
		fmt.Printf("\nMuvaffaqiyatli yakunlandi!\nYangi fayl nomi: %s\n", filepath.Base(outputFile))
	}

	fmt.Println("\nDasturni yopish uchun Enter tugmasini bosing...")
	fmt.Scanln()
}
