package sheets_service

// Cell construction: colors, formats, and the mk* factories every
// sheet row is built from.

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"register/pkg/models"

	"google.golang.org/api/sheets/v4"
)

var cellColors = map[string]*sheets.Color{
	models.ColorBlack: {
		Alpha: 1,
		Blue:  0,
		Red:   0,
		Green: 0,
	},
	models.ColorWhite: {
		Alpha: 1,
		Blue:  1,
		Red:   1,
		Green: 1,
	},
	models.ColorGreen: {
		Alpha: 1,
		Blue:  0,
		Red:   0.5,
		Green: 1,
	},
	models.ColorYellow: {
		Alpha: 1,
		Blue:  0.6,
		Red:   1,
		Green: 1,
	},
	models.ColorBlue: {
		Alpha: 1,
		Blue:  1,
		Red:   0,
		Green: 0.8,
	},
	models.ColorLightGrey: {
		Alpha: 1,
		Blue:  0.937,
		Red:   0.937,
		Green: 0.937,
	},
	models.ColorGrey: {
		Alpha: 1,
		Blue:  0.8,
		Red:   0.8,
		Green: 0.8,
	},
}

func getBackgroundColor(trans *models.Transaction, income bool) string {
	if income {
		return models.ColorGreen
	} else if trans.TaxDeductible {
		return models.ColorYellow
	}
	return models.ColorWhite
}

func getOddOrEvenRowColor(i int, even, odd string) string {
	if i%2 == 0 {
		return even
	}
	return odd
}

func mkColor(name string) *sheets.Color {
	return cellColors[name]
}

func mkBorders(on bool) *sheets.Borders {
	if !on {
		return &sheets.Borders{}
	}
	return &sheets.Borders{
		Left: &sheets.Border{
			Color: mkColor(models.ColorBlack),
			Style: "SOLID",
		},
		Right: &sheets.Border{
			Color: mkColor(models.ColorBlack),
			Style: "SOLID",
		},
		Bottom: &sheets.Border{
			Color: mkColor(models.ColorBlack),
			Style: "SOLID",
		},
	}
}

func mkTextFormat() *sheets.TextFormat {
	return &sheets.TextFormat{
		FontFamily: "Arial",
		FontSize:   10,
	}
}

func mkTextFormatReconcileColumn() *sheets.TextFormat {
	return &sheets.TextFormat{
		FontFamily: "Archivo Black",
		FontSize:   12,
	}
}

func mkNumberFormatDate() *sheets.NumberFormat {
	return &sheets.NumberFormat{
		Pattern: "mm/dd/yy",
		Type:    "DATE",
	}
}

func mkCellFormat(align, colorName string, bordersOn bool) *sheets.CellFormat {
	return &sheets.CellFormat{
		HorizontalAlignment: strings.ToUpper(align),
		TextFormat:          mkTextFormat(),
		BackgroundColor:     mkColor(colorName),
		Borders:             mkBorders(bordersOn),
	}
}

func mkCellDataString(value, align, color string, borders bool) *sheets.CellData {
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			StringValue: &value,
		},
		UserEnteredFormat: mkCellFormat(align, color, borders),
	}
}

func mkBoldFormat(value, align, color string, borders bool) *sheets.CellData {
	c := mkCellDataString(value, align, color, borders)
	c.UserEnteredFormat.TextFormat.Bold = true
	return c
}

func mkCellDataNumber(value float64, align, color string, borders bool) *sheets.CellData {
	v := math.Round(value*100) / 100
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			NumberValue: &v,
		},
		UserEnteredFormat: mkCellFormat(align, color, borders),
	}
}

func mkCellDataDollars(value float64, align, colorName string, borders bool) *sheets.CellData {
	v := math.Round(value*100) / 100
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			NumberValue: &v,
		},
		UserEnteredFormat: &sheets.CellFormat{
			HorizontalAlignment: strings.ToUpper(align),
			TextFormat:          mkTextFormat(),
			NumberFormat:        getNumberFormatDollars(),
			BackgroundColor:     mkColor(colorName),
			Borders:             mkBorders(borders),
		},
	}
}

func mkCellDataEmpty(align, colorName string, borders bool) *sheets.CellData {
	s := ""
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			StringValue: &s,
		},
		UserEnteredFormat: &sheets.CellFormat{
			HorizontalAlignment: strings.ToUpper(align),
			TextFormat:          mkTextFormat(),
			NumberFormat:        getNumberFormatDollars(),
			BackgroundColor:     mkColor(colorName),
			Borders:             mkBorders(borders),
		},
	}
}

func mkCellDataFormula(value string, align, colorName string, borders bool) *sheets.CellData {
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			FormulaValue: &value,
		},
		UserEnteredFormat: &sheets.CellFormat{
			HorizontalAlignment: strings.ToUpper(align),
			TextFormat:          mkTextFormat(),
			NumberFormat:        getNumberFormatDollars(),
			BackgroundColor:     mkColor(colorName),
			Borders:             mkBorders(borders),
		},
	}
}

func getCellDataReconcileColumn(value, align, colorName string, borders bool) *sheets.CellData {
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			StringValue: &value,
		},
		UserEnteredFormat: &sheets.CellFormat{
			HorizontalAlignment: strings.ToUpper(align),
			TextFormat:          mkTextFormatReconcileColumn(),
			BackgroundColor:     mkColor(colorName),
			Borders:             mkBorders(borders),
		},
	}
}

func getNumberFormatDollars() *sheets.NumberFormat {
	return &sheets.NumberFormat{
		Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
		Type:    "CURRENCY",
	}
}

func getCellDataDate(dateString, align, colorName string, borders bool) (*sheets.CellData, error) {
	dateString = formatYear(dateString)
	csvTime, err := time.Parse("01/02/06", dateString)
	if err != nil {
		return nil, fmt.Errorf("could not parse date string: %s: %s", dateString, err.Error())
	}

	serialTime, _ := time.Parse("01/02/2006", "12/30/1899")
	serialFormatFloat, err := strconv.ParseFloat(fmt.Sprintf("%.0f.0", csvTime.Sub(serialTime).Hours()/24.0), 64)
	if err != nil {
		return nil, fmt.Errorf("could not create serial format float with serialTime=%+v, csvTime=%+v: %s",
			serialTime, csvTime, err.Error())
	}

	cell := &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			NumberValue: &serialFormatFloat,
		},
		UserEnteredFormat: &sheets.CellFormat{
			HorizontalAlignment: strings.ToUpper(align),
			TextFormat:          mkTextFormat(),
			NumberFormat:        mkNumberFormatDate(),
			BackgroundColor:     mkColor(colorName),
			Borders:             mkBorders(borders),
		},
	}
	return cell, nil
}
