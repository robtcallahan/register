package cmd

import (
	"register/pkg/models"
	"register/pkg/repository"
)

// fakeQueryRepo is a configurable in-memory QueryRepo for cmd tests. Each
// method returns its configured value (or error); nothing touches a live DB.
type fakeQueryRepo struct {
	columns           []models.Column
	columnsErr        error
	columnsCalls      int
	columnsErrOnCall  int // fail GetColumns on this 1-based call (0 = never)
	lookupData        []*models.DataRow
	lookupErr         error
	maxPriority       int
	maxPriorityErr    error
	createMerchantErr error
	transactions      []models.Transaction
	transactionsErr   error
	updateErr         error
	merchants         []models.Merchant
	merchantsErr      error
	nameMap           map[string]string
	nameMapErr        error
}

var _ repository.QueryRepo = (*fakeQueryRepo)(nil)

func (f *fakeQueryRepo) GetColumns() ([]models.Column, error) {
	f.columnsCalls++
	if f.columnsErrOnCall > 0 {
		if f.columnsCalls == f.columnsErrOnCall {
			return nil, f.columnsErr
		}
		return f.columns, nil
	}
	return f.columns, f.columnsErr
}

func (f *fakeQueryRepo) AddColumn(col *models.Column) error {
	return nil
}

func (f *fakeQueryRepo) RenameColumn(id int, name string) error {
	return nil
}

func (f *fakeQueryRepo) DeleteColumn(id int) error {
	return nil
}

func (f *fakeQueryRepo) MoveColumn(id int, newIndex int) error {
	return nil
}

func (f *fakeQueryRepo) GetMerchantsByColumn(columnID int) ([]models.Merchant, error) {
	return f.merchants, f.merchantsErr
}

func (f *fakeQueryRepo) CreateMerchant(m *models.Merchant) error {
	return f.createMerchantErr
}

func (f *fakeQueryRepo) GetMaxMerchantPriority() (int, error) {
	return f.maxPriority, f.maxPriorityErr
}

func (f *fakeQueryRepo) GetTransactions() ([]models.Transaction, error) {
	return f.transactions, f.transactionsErr
}

func (f *fakeQueryRepo) UpdateTransactionTables(trans []*models.Transaction) error {
	return f.updateErr
}

func (f *fakeQueryRepo) GetLookupData() ([]*models.DataRow, error) {
	return f.lookupData, f.lookupErr
}

func (f *fakeQueryRepo) GetNameMapToColumn() (map[string]string, error) {
	return f.nameMap, f.nameMapErr
}
