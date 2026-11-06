package sheets_service

import (
	"strings"
	"testing"

	"register/pkg/models"
)

func Test_registerService_InsertRegisterColumn_Errors(t *testing.T) {
	sheet := RegisterSheet{
		ID:          617336355,
		TabName:     RegisterTabName,
		SheetCoords: SheetCoords{RowCount: 5},
	}
	t.Run("batch update error", func(t *testing.T) {
		svc := NewRegisterService(&stubSheetsProvider{batchUpdateErr: errTestProvider})
		if err := svc.InsertRegisterColumn(sheet, 3, "NewCol"); err == nil || !strings.Contains(err.Error(), "could not insert column") {
			t.Errorf("InsertRegisterColumn() error = %v, want insert error", err)
		}
	})
	t.Run("write error", func(t *testing.T) {
		svc := NewRegisterService(&stubSheetsProvider{updateErr: errTestProvider})
		// Index 3 reads the neighbor column's formulas first; the stub
		// returns no formulas, so the write is what fails.
		if err := svc.InsertRegisterColumn(sheet, 3, "NewCol"); err == nil || !strings.Contains(err.Error(), "could not write column C") {
			t.Errorf("InsertRegisterColumn() error = %v, want write error", err)
		}
	})
}

func Test_registerService_DeleteRegisterColumn_BatchUpdateError(t *testing.T) {
	sheet := RegisterSheet{ID: 617336355, TabName: RegisterTabName}
	svc := NewRegisterService(&stubSheetsProvider{batchUpdateErr: errTestProvider})
	if err := svc.DeleteRegisterColumn(sheet, 12); err == nil || !strings.Contains(err.Error(), "could not delete column L") {
		t.Errorf("DeleteRegisterColumn() error = %v, want delete error", err)
	}
}

func Test_registerService_MoveRegisterColumn_BatchUpdateError(t *testing.T) {
	sheet := RegisterSheet{ID: 617336355, TabName: RegisterTabName}
	svc := NewRegisterService(&stubSheetsProvider{batchUpdateErr: errTestProvider})
	if err := svc.MoveRegisterColumn(sheet, 11, 12); err == nil || !strings.Contains(err.Error(), "could not move column K") {
		t.Errorf("MoveRegisterColumn() error = %v, want move error", err)
	}
}

func TestPlanColumnMove_UnknownAfter(t *testing.T) {
	if _, _, err := PlanColumnMove(moveTestColumns(), 13, 99); err == nil || !strings.Contains(err.Error(), "no column at index 99") {
		t.Errorf("PlanColumnMove(13, 99) error = %v, want unknown-after error", err)
	}
}

func Test_colorOrNone(t *testing.T) {
	if got := colorOrNone(""); got != "uncolored" {
		t.Errorf("colorOrNone(%q) = %q, want uncolored", "", got)
	}
	if got := colorOrNone(models.ColorGreen); got != models.ColorGreen {
		t.Errorf("colorOrNone(%q) = %q, want %q", models.ColorGreen, got, models.ColorGreen)
	}
}
