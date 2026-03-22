package service

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

func BuildTransactionHistoryCSV(items []TransactionHistoryItem) ([]byte, error) {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	header := []string{"id", "created_at", "toko_name", "player", "code", "type", "status", "reference", "amount", "netto"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("write csv header: %w", err)
	}

	for _, item := range items {
		row := []string{
			strconv.FormatUint(item.ID, 10),
			item.CreatedAt,
			item.TokoName,
			item.Player,
			item.Code,
			item.Type,
			item.Status,
			item.Reference,
			strconv.FormatUint(item.Amount, 10),
			strconv.FormatUint(item.Netto, 10),
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("write csv row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush csv: %w", err)
	}
	return buffer.Bytes(), nil
}

func BuildTransactionHistoryDOCX(items []TransactionHistoryItem) ([]byte, error) {
	buffer := &bytes.Buffer{}
	zipWriter := zip.NewWriter(buffer)

	if err := writeZipFile(zipWriter, "[Content_Types].xml", contentTypesXML); err != nil {
		return nil, err
	}
	if err := writeZipFile(zipWriter, "_rels/.rels", rootRelationshipsXML); err != nil {
		return nil, err
	}
	if err := writeZipFile(zipWriter, "word/document.xml", renderDocumentXML(items)); err != nil {
		return nil, err
	}
	if err := writeZipFile(zipWriter, "word/_rels/document.xml.rels", documentRelationshipsXML); err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close docx archive: %w", err)
	}
	return buffer.Bytes(), nil
}

func writeZipFile(writer *zip.Writer, path string, content string) error {
	fileWriter, err := writer.Create(path)
	if err != nil {
		return fmt.Errorf("create docx part %s: %w", path, err)
	}
	if _, err := fileWriter.Write([]byte(content)); err != nil {
		return fmt.Errorf("write docx part %s: %w", path, err)
	}
	return nil
}

func renderDocumentXML(items []TransactionHistoryItem) string {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	builder.WriteString(`<w:document xmlns:wpc="http://schemas.microsoft.com/office/word/2010/wordprocessingCanvas" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006" xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math" xmlns:v="urn:schemas-microsoft-com:vml" xmlns:wp14="http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:w10="urn:schemas-microsoft-com:office:word" xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:w14="http://schemas.microsoft.com/office/word/2010/wordml" xmlns:w15="http://schemas.microsoft.com/office/word/2012/wordml" mc:Ignorable="w14 w15"><w:body>`)
	builder.WriteString(`<w:p><w:r><w:t>Transaction History Export</w:t></w:r></w:p>`)
	builder.WriteString(`<w:tbl>`)
	builder.WriteString(`<w:tr>`)
	docxCell(&builder, "ID")
	docxCell(&builder, "Created At")
	docxCell(&builder, "Toko")
	docxCell(&builder, "Player")
	docxCell(&builder, "Code")
	docxCell(&builder, "Type")
	docxCell(&builder, "Status")
	docxCell(&builder, "Reference")
	docxCell(&builder, "Amount")
	docxCell(&builder, "Netto")
	builder.WriteString(`</w:tr>`)

	for _, item := range items {
		builder.WriteString(`<w:tr>`)
		docxCell(&builder, strconv.FormatUint(item.ID, 10))
		docxCell(&builder, item.CreatedAt)
		docxCell(&builder, item.TokoName)
		docxCell(&builder, item.Player)
		docxCell(&builder, item.Code)
		docxCell(&builder, item.Type)
		docxCell(&builder, item.Status)
		docxCell(&builder, item.Reference)
		docxCell(&builder, strconv.FormatUint(item.Amount, 10))
		docxCell(&builder, strconv.FormatUint(item.Netto, 10))
		builder.WriteString(`</w:tr>`)
	}

	builder.WriteString(`</w:tbl>`)
	builder.WriteString(`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr>`)
	builder.WriteString(`</w:body></w:document>`)
	return builder.String()
}

func docxCell(builder *strings.Builder, value string) {
	builder.WriteString(`<w:tc><w:p><w:r><w:t>`)
	builder.WriteString(escapeXML(value))
	builder.WriteString(`</w:t></w:r></w:p></w:tc>`)
}

func escapeXML(value string) string {
	var buffer bytes.Buffer
	if err := xml.EscapeText(&buffer, []byte(value)); err != nil {
		return value
	}
	return buffer.String()
}

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`

const rootRelationshipsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

const documentRelationshipsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`
