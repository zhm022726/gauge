// Copyright 2015 ThoughtWorks, Inc.

// This file is part of Gauge.

// Gauge is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Gauge is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with Gauge.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"fmt"
	"github.com/getgauge/common"
	"github.com/getgauge/gauge/gauge_messages"
	"github.com/getgauge/gauge/parser"
	"sort"
	"strings"
)

const (
	TABLE_LEFT_SPACING = 5
)

func formatSpecFiles(specFiles ...string) []*parser.ParseResult {
	specs, results := parser.ParseSpecFiles(specFiles, &parser.ConceptDictionary{})
	for i, spec := range specs {
		if err := formatAndSave(spec); err != nil {
			results[i].ParseError = &parser.ParseError{Message: err.Error()}
		}
	}
	return results
}

func getRepeatedChars(character string, repeatCount int) string {
	formatted := ""
	for i := 0; i < repeatCount; i++ {
		formatted = fmt.Sprintf("%s%s", formatted, character)
	}
	return formatted
}

func formatSpecHeading(specHeading string) string {
	return formatHeading(specHeading, "=")
}

func formatScenarioHeading(scenarioHeading string) string {
	return fmt.Sprintf("%s", formatHeading(scenarioHeading, "-"))
}

func formatStep(step *parser.Step) string {
	text := step.Value()
	paramCount := strings.Count(text, parser.ParameterPlaceholder)
	for i := 0; i < paramCount; i++ {
		argument := step.Args()[i]
		formattedArg := ""
		if argument.ArgType == parser.TableArg {
			formattedTable := formatTable(&argument.Table)
			formattedArg = fmt.Sprintf("\n%s", formattedTable)
		} else if argument.ArgType == parser.Dynamic {
			formattedArg = fmt.Sprintf("<%s>", parser.GetUnescapedString(argument.Value))
		} else if argument.ArgType == parser.SpecialString || argument.ArgType == parser.SpecialTable {
			formattedArg = fmt.Sprintf("<%s>", parser.GetUnescapedString(argument.Name))
		} else {
			formattedArg = fmt.Sprintf("\"%s\"", parser.GetUnescapedString(argument.Value))
		}
		text = strings.Replace(text, parser.ParameterPlaceholder, formattedArg, 1)
	}
	stepText := ""
	if strings.HasSuffix(text, "\n") {
		stepText = fmt.Sprintf("* %s", text)
	} else {
		stepText = fmt.Sprintf("* %s\n", text)
	}
	return stepText
}

func formatConcept(protoConcept *gauge_messages.ProtoConcept) string {
	conceptText := "* "
	for _, fragment := range protoConcept.ConceptStep.GetFragments() {
		if fragment.GetFragmentType() == gauge_messages.Fragment_Text {
			conceptText = conceptText + fragment.GetText()
		} else if fragment.GetFragmentType() == gauge_messages.Fragment_Parameter {
			if fragment.GetParameter().GetParameterType() == (gauge_messages.Parameter_Table | gauge_messages.Parameter_Special_Table) {
				conceptText += "\n" + formatTable(parser.TableFrom(fragment.GetParameter().GetTable()))
			} else {
				conceptText = conceptText + "\"" + fragment.GetParameter().GetValue() + "\""
			}
		}
	}
	return conceptText + "\n"
}

func formatHeading(heading, headingChar string) string {
	trimmedHeading := strings.TrimSpace(heading)
	length := len(trimmedHeading)
	return fmt.Sprintf("%s\n%s\n", trimmedHeading, getRepeatedChars(headingChar, length))
}

func formatTable(table *parser.Table) string {
	columnToWidthMap := make(map[int]int)
	for i, header := range table.Headers {
		//table.get(header) returns a list of cells in that particular column
		cells := table.Get(header)
		columnToWidthMap[i] = findLongestCellWidth(cells, len(header))
	}

	var tableStringBuffer bytes.Buffer
	tableStringBuffer.WriteString(fmt.Sprintf("%s|", getRepeatedChars(" ", TABLE_LEFT_SPACING)))
	for i, header := range table.Headers {
		width := columnToWidthMap[i]
		tableStringBuffer.WriteString(fmt.Sprintf("%s|", addPaddingToCell(header, width)))
	}

	tableStringBuffer.WriteString("\n")
	tableStringBuffer.WriteString(fmt.Sprintf("%s|", getRepeatedChars(" ", TABLE_LEFT_SPACING)))
	for i, _ := range table.Headers {
		width := columnToWidthMap[i]
		cell := getRepeatedChars("-", width)
		tableStringBuffer.WriteString(fmt.Sprintf("%s|", addPaddingToCell(cell, width)))
	}

	tableStringBuffer.WriteString("\n")
	for _, row := range table.Rows() {
		tableStringBuffer.WriteString(fmt.Sprintf("%s|", getRepeatedChars(" ", TABLE_LEFT_SPACING)))
		for i, cell := range row {
			width := columnToWidthMap[i]
			tableStringBuffer.WriteString(fmt.Sprintf("%s|", addPaddingToCell(cell, width)))
		}
		tableStringBuffer.WriteString("\n")
	}

	return string(tableStringBuffer.Bytes())
}

func addPaddingToCell(cellValue string, width int) string {
	padding := getRepeatedChars(" ", width-len(cellValue))
	return fmt.Sprintf("%s%s", cellValue, padding)
}

func findLongestCellWidth(columnCells []parser.TableCell, minValue int) int {
	longestLength := minValue
	for _, cellValue := range columnCells {
		cellValueLen := len(cellValue.GetValue())
		if cellValueLen > longestLength {
			longestLength = cellValueLen
		}
	}
	return longestLength
}

func formatComment(comment *parser.Comment) string {
	if comment.Value() == "\n" {
		return comment.Value()
	}
	return fmt.Sprintf("%s\n", comment.Value())
}

func formatTags(tags *parser.Tags) string {
	if tags == nil || len(tags.Values()) == 0 {
		return ""
	}
	var b bytes.Buffer
	b.WriteString("tags: ")
	for i, tag := range tags.Values() {
		b.WriteString(tag)
		if (i + 1) != len(tags.Values()) {
			b.WriteString(", ")
		}
	}
	b.WriteString("\n")
	return string(b.Bytes())
}

func formatExternalDataTable(dataTable *parser.DataTable) string {
	if dataTable == nil || len(dataTable.Value()) == 0 {
		return ""
	}
	var b bytes.Buffer
	b.WriteString(dataTable.Value())
	b.WriteString("\n")
	return string(b.Bytes())
}

func formatAndSave(spec *parser.Specification) error {
	formatted := formatSpecification(spec)
	if err := common.SaveFile(spec.FileName, formatted, true); err != nil {
		return err
	}
	return nil
}

func formatSpecification(specification *parser.Specification) string {
	var formattedSpec bytes.Buffer
	formatter := &formatter{buffer: formattedSpec}
	specification.Traverse(formatter)
	return string(formatter.buffer.Bytes())
}

type ByLineNo []*parser.Concept

func (s ByLineNo) Len() int {
	return len(s)
}

func (s ByLineNo) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ByLineNo) Less(i, j int) bool {
	return s[i].conceptStep.lineNo < s[j].conceptStep.lineNo
}

func sortConcepts(conceptDictionary *parser.ConceptDictionary, conceptMap map[string]string) []*parser.Concept {
	concepts := make([]*parser.Concept, 0)
	for _, concept := range conceptDictionary.conceptsMap {
		conceptMap[concept.fileName] = ""
		concepts = append(concepts, concept)
	}
	sort.Sort(ByLineNo(concepts))
	return concepts
}

func formatConceptSteps(conceptMap map[string]string, concept *parser.Concept) {
	conceptMap[concept.fileName] += strings.TrimSpace(strings.Replace(formatStep(concept.conceptStep), "*", "#", 1)) + "\n"
	for i := 1; i < len(concept.conceptStep.items); i++ {
		conceptMap[concept.fileName] += formatItem(concept.conceptStep.items[i])
	}
}

func formatConcepts(conceptDictionary *parser.ConceptDictionary) map[string]string {
	conceptMap := make(map[string]string)
	for _, concept := range sortConcepts(conceptDictionary, conceptMap) {
		for _, comment := range concept.conceptStep.preComments {
			conceptMap[concept.fileName] += formatComment(comment)
		}
		formatConceptSteps(conceptMap, concept)
	}
	return conceptMap
}

func formatItem(item parser.Item) string {
	switch item.kind() {
	case commentKind:
		comment := item.(*comment)
		if comment.value == "\n" {
			return comment.value
		}
		return fmt.Sprintf("%s\n", comment.value)
	case stepKind:
		step := item.(*step)
		return formatStep(step)
	case tableKind:
		table := item.(*table)
		return formatTable(table)
	case tagKind:
		tags := item.(*tags)
		return formatTags(tags)
	}
	return ""
}