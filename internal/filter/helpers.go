package filter

import (
	"github.com/lugondev/go-carbon/internal/datasource"
	"github.com/lugondev/go-carbon/pkg/types"
)

func CheckAccountFilters(datasourceID datasource.DatasourceID, filters []Filter, metadata *AccountMetadata, account *types.Account) bool {
	for _, f := range filters {
		if !f.FilterAccount(datasourceID, metadata, account) {
			return false
		}
	}
	return true
}

func CheckInstructionFilters(datasourceID datasource.DatasourceID, filters []Filter, nestedInstruction NestedInstruction) bool {
	for _, f := range filters {
		if !f.FilterInstruction(datasourceID, nestedInstruction) {
			return false
		}
	}
	return true
}

func CheckTransactionFilters(datasourceID datasource.DatasourceID, filters []Filter, txMetadata TransactionMetadata, nestedInstructions NestedInstructions) bool {
	for _, f := range filters {
		if !f.FilterTransaction(datasourceID, txMetadata, nestedInstructions) {
			return false
		}
	}
	return true
}

func CheckAccountDeletionFilters(datasourceID datasource.DatasourceID, filters []Filter, accountDeletion *datasource.AccountDeletion) bool {
	for _, f := range filters {
		if !f.FilterAccountDeletion(datasourceID, accountDeletion) {
			return false
		}
	}
	return true
}

func CheckBlockDetailsFilters(datasourceID datasource.DatasourceID, filters []Filter, blockDetails *datasource.BlockDetails) bool {
	for _, f := range filters {
		if !f.FilterBlockDetails(datasourceID, blockDetails) {
			return false
		}
	}
	return true
}
