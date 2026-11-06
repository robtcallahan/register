package sheets_service

import (
	"reflect"
	"testing"

	"register/pkg/models"
)

func Test_getRowValues_OutOfRange(t *testing.T) {
	values := [][]any{{"a"}, {"b"}}
	if got := getRowValues(values, 2); got != nil {
		t.Errorf("getRowValues(values, 2) = %v, want nil", got)
	}
	if got := getRowValues(values, -1); got != nil {
		t.Errorf("getRowValues(values, -1) = %v, want nil", got)
	}
	if got := getRowValues(values, 1); !reflect.DeepEqual(got, []any{"b"}) {
		t.Errorf("getRowValues(values, 1) = %v, want [b]", got)
	}
	if got := getRowValues(nil, 0); got != nil {
		t.Errorf("getRowValues(nil, 0) = %v, want nil", got)
	}
}

func Test_getSourceField_EmptySource(t *testing.T) {
	values := []any{"", "", "01/02/2023", "Rent"}
	if got := getSourceField(values); got != "wellsfargo" {
		t.Errorf("getSourceField() = %q, want wellsfargo", got)
	}
}

func Test_getAmountString_Branches(t *testing.T) {
	mk := func(withdrawal, deposit, credit string) []any {
		values := make([]any, CreditPurchase+1)
		for i := range values {
			values[i] = ""
		}
		values[Withdrawals] = withdrawal
		values[Deposits] = deposit
		values[CreditPurchase] = credit
		return values
	}
	tests := []struct {
		name   string
		values []any
		want   string
	}{
		{"bare dash", mk("-", "", ""), "0.00"},
		{"empty everywhere", mk("", "", ""), "0.00"},
		{"unparseable", mk("abc", "", ""), "0.00"},
		{"parenthesized negative", mk("(10.00)", "", ""), "-10.00"},
		{"parenthesized negative with dollar sign", mk("$(10.00)", "", ""), "-10.00"},
		{"zero", mk("0", "", ""), "0.00"},
		{"negative zero normalizes", mk("-0.00", "", ""), "0.00"},
		{"deposit parenthesized negative", mk("", "(5.50)", ""), "-5.50"},
		{"credit purchase", mk("", "", "30.00"), "30.00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAmountString(tt.values); got != tt.want {
				t.Errorf("getAmountString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_getDollarsCellByIndex_OutOfRange(t *testing.T) {
	values := []any{"$ 10.20 "}
	if got, err := getDollarsCellByIndex(values, 1); err != nil || got != 0 {
		t.Errorf("getDollarsCellByIndex(values, 1) = (%v, %v), want (0, nil)", got, err)
	}
	if got, err := getDollarsCellByIndex(values, -1); err != nil || got != 0 {
		t.Errorf("getDollarsCellByIndex(values, -1) = (%v, %v), want (0, nil)", got, err)
	}
}

func Test_newRowDecoder_DuplicateHeader(t *testing.T) {
	if _, err := newRowDecoder([]any{"Weekly", " weekly "}); err == nil {
		t.Errorf("newRowDecoder() error = nil, want duplicate header error")
	}
	decoder, err := newRowDecoder([]any{"", "Weekly", ""})
	if err != nil {
		t.Fatalf("newRowDecoder() error = %v", err)
	}
	if got, err := decoder.require("Weekly"); err != nil || got != 1 {
		t.Errorf("require(Weekly) = (%d, %v), want (1, nil)", got, err)
	}
}

func Test_rowDecoder_CellShortRow(t *testing.T) {
	decoder, err := newRowDecoder([]any{"A", "B", "C"})
	if err != nil {
		t.Fatalf("newRowDecoder() error = %v", err)
	}
	if got := decoder.cell([]any{"x"}, "C"); got != "" {
		t.Errorf("cell(short row, C) = %v, want empty", got)
	}
	if got := decoder.cell([]any{"x", "y", "z"}, "B"); got != "y" {
		t.Errorf("cell(row, B) = %v, want y", got)
	}
	if got := decoder.cell([]any{"x", "y", "z"}, "Missing"); got != "" {
		t.Errorf("cell(row, Missing) = %v, want empty", got)
	}
}

func Test_getBackgroundColor_TaxDeductible(t *testing.T) {
	if got := getBackgroundColor(&models.Transaction{TaxDeductible: true}, false); got != models.ColorYellow {
		t.Errorf("getBackgroundColor(tax deductible) = %q, want %q", got, models.ColorYellow)
	}
	// Income wins over the tax-deductible color.
	if got := getBackgroundColor(&models.Transaction{TaxDeductible: true}, true); got != models.ColorGreen {
		t.Errorf("getBackgroundColor(income, tax deductible) = %q, want %q", got, models.ColorGreen)
	}
}
