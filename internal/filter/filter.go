// Package filter provides a flexible filtering system for the carbon pipeline.
//
// Filters allow you to selectively process updates based on various criteria.
// They can be applied to different types of updates (accounts, instructions,
// transactions, account deletions, and block details) and can filter based on
// datasource IDs, update content, or any other custom logic.
package filter

import (
	"github.com/lugondev/go-carbon/internal/datasource"
	"github.com/lugondev/go-carbon/pkg/types"
)

// AccountMetadata holds metadata for an account update.
type AccountMetadata struct {
	// Slot is the Solana slot number where the account was updated.
	Slot uint64

	// Pubkey is the public key of the account.
	Pubkey types.Pubkey

	// TransactionSignature is the signature of the transaction that caused the update.
	TransactionSignature *types.Signature
}

// TransactionMetadata contains metadata about a transaction.
// This is a forward declaration; the full type is in the transaction package.
type TransactionMetadata interface {
	GetSlot() uint64
	GetSignature() types.Signature
	GetFeePayer() types.Pubkey
}

// NestedInstruction represents an instruction with potential nested inner instructions.
// This is a forward declaration; the full type is in the instruction package.
type NestedInstruction interface {
	GetProgramID() types.Pubkey
	GetData() []byte
}

// NestedInstructions is a collection of nested instructions.
type NestedInstructions interface {
	Len() int
	Get(index int) NestedInstruction
}

// Filter defines the interface for filtering updates in the carbon pipeline.
//
// Filters allow you to selectively process updates based on various criteria.
// Each filter method returns true if the update should be processed, or
// false if it should be skipped.
type Filter interface {
	// FilterAccount filters account updates.
	// Returns true if the account update should be processed.
	FilterAccount(
		datasourceID datasource.DatasourceID,
		accountMetadata *AccountMetadata,
		account *types.Account,
	) bool

	// FilterInstruction filters instruction updates.
	// Returns true if the instruction update should be processed.
	FilterInstruction(
		datasourceID datasource.DatasourceID,
		nestedInstruction NestedInstruction,
	) bool

	// FilterTransaction filters transaction updates.
	// Returns true if the transaction update should be processed.
	FilterTransaction(
		datasourceID datasource.DatasourceID,
		transactionMetadata TransactionMetadata,
		nestedInstructions NestedInstructions,
	) bool

	// FilterAccountDeletion filters account deletion updates.
	// Returns true if the account deletion update should be processed.
	FilterAccountDeletion(
		datasourceID datasource.DatasourceID,
		accountDeletion *datasource.AccountDeletion,
	) bool

	// FilterBlockDetails filters block details updates.
	// Returns true if the block details update should be processed.
	FilterBlockDetails(
		datasourceID datasource.DatasourceID,
		blockDetails *datasource.BlockDetails,
	) bool
}

// BaseFilter provides default implementations that allow all updates.
// Embed this in your filter implementations to only override the methods you need.
type BaseFilter struct{}

func (f *BaseFilter) FilterAccount(datasourceID datasource.DatasourceID, accountMetadata *AccountMetadata, account *types.Account) bool {
	return true
}

func (f *BaseFilter) FilterInstruction(datasourceID datasource.DatasourceID, nestedInstruction NestedInstruction) bool {
	return true
}

func (f *BaseFilter) FilterTransaction(datasourceID datasource.DatasourceID, transactionMetadata TransactionMetadata, nestedInstructions NestedInstructions) bool {
	return true
}

func (f *BaseFilter) FilterAccountDeletion(datasourceID datasource.DatasourceID, accountDeletion *datasource.AccountDeletion) bool {
	return true
}

func (f *BaseFilter) FilterBlockDetails(datasourceID datasource.DatasourceID, blockDetails *datasource.BlockDetails) bool {
	return true
}

// DatasourceFilter filters updates based on their datasource ID.
// Only updates from allowed datasources will be processed.
type DatasourceFilter struct {
	BaseFilter
	allowedDatasources []datasource.DatasourceID
}

// NewDatasourceFilter creates a new filter that allows updates from a single datasource.
func NewDatasourceFilter(datasourceID datasource.DatasourceID) *DatasourceFilter {
	return &DatasourceFilter{
		allowedDatasources: []datasource.DatasourceID{datasourceID},
	}
}

// NewDatasourceFilterMany creates a new filter that allows updates from multiple datasources.
func NewDatasourceFilterMany(datasourceIDs []datasource.DatasourceID) *DatasourceFilter {
	return &DatasourceFilter{
		allowedDatasources: datasourceIDs,
	}
}

// isAllowed checks if the datasource ID is in the allowed list.
func (f *DatasourceFilter) isAllowed(id datasource.DatasourceID) bool {
	for _, allowed := range f.allowedDatasources {
		if allowed.Equals(id) {
			return true
		}
	}
	return false
}

func (f *DatasourceFilter) FilterAccount(datasourceID datasource.DatasourceID, accountMetadata *AccountMetadata, account *types.Account) bool {
	return f.isAllowed(datasourceID)
}

func (f *DatasourceFilter) FilterInstruction(datasourceID datasource.DatasourceID, nestedInstruction NestedInstruction) bool {
	return f.isAllowed(datasourceID)
}

func (f *DatasourceFilter) FilterTransaction(datasourceID datasource.DatasourceID, transactionMetadata TransactionMetadata, nestedInstructions NestedInstructions) bool {
	return f.isAllowed(datasourceID)
}

func (f *DatasourceFilter) FilterAccountDeletion(datasourceID datasource.DatasourceID, accountDeletion *datasource.AccountDeletion) bool {
	return f.isAllowed(datasourceID)
}

func (f *DatasourceFilter) FilterBlockDetails(datasourceID datasource.DatasourceID, blockDetails *datasource.BlockDetails) bool {
	return f.isAllowed(datasourceID)
}

// AllowAllFilter is a filter that allows all updates to pass through.
type AllowAllFilter struct {
	BaseFilter
}

// NewAllowAllFilter creates a new filter that allows all updates.
func NewAllowAllFilter() *AllowAllFilter {
	return &AllowAllFilter{}
}

// ApplyFilters applies all filters to a filter function and returns true only if all pass.
func ApplyFilters[T any](filters []Filter, datasourceID datasource.DatasourceID, data T, filterFunc func(Filter, datasource.DatasourceID, T) bool) bool {
	for _, f := range filters {
		if !filterFunc(f, datasourceID, data) {
			return false
		}
	}
	return true
}

// FilterChain chains multiple filters together.
// All filters must pass for the update to be processed.
type FilterChain struct {
	filters []Filter
}

// NewFilterChain creates a new filter chain with the given filters.
func NewFilterChain(filters ...Filter) *FilterChain {
	return &FilterChain{filters: filters}
}

// Add adds a filter to the chain.
func (c *FilterChain) Add(f Filter) {
	c.filters = append(c.filters, f)
}

func (c *FilterChain) FilterAccount(datasourceID datasource.DatasourceID, accountMetadata *AccountMetadata, account *types.Account) bool {
	for _, f := range c.filters {
		if !f.FilterAccount(datasourceID, accountMetadata, account) {
			return false
		}
	}
	return true
}

func (c *FilterChain) FilterInstruction(datasourceID datasource.DatasourceID, nestedInstruction NestedInstruction) bool {
	for _, f := range c.filters {
		if !f.FilterInstruction(datasourceID, nestedInstruction) {
			return false
		}
	}
	return true
}

func (c *FilterChain) FilterTransaction(datasourceID datasource.DatasourceID, transactionMetadata TransactionMetadata, nestedInstructions NestedInstructions) bool {
	for _, f := range c.filters {
		if !f.FilterTransaction(datasourceID, transactionMetadata, nestedInstructions) {
			return false
		}
	}
	return true
}

func (c *FilterChain) FilterAccountDeletion(datasourceID datasource.DatasourceID, accountDeletion *datasource.AccountDeletion) bool {
	for _, f := range c.filters {
		if !f.FilterAccountDeletion(datasourceID, accountDeletion) {
			return false
		}
	}
	return true
}

func (c *FilterChain) FilterBlockDetails(datasourceID datasource.DatasourceID, blockDetails *datasource.BlockDetails) bool {
	for _, f := range c.filters {
		if !f.FilterBlockDetails(datasourceID, blockDetails) {
			return false
		}
	}
	return true
}
