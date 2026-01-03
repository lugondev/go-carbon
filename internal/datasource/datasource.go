// Package datasource provides traits and structures for managing and consuming
// data updates from various sources.
//
// The datasource package defines the Datasource interface and associated data types
// for handling updates related to accounts, transactions, and account deletions.
// This allows for flexible data ingestion from various Solana data sources,
// enabling integration with the carbon processing pipeline.
package datasource

import (
	"context"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/lugondev/go-carbon/internal/metrics"
	"github.com/lugondev/go-carbon/pkg/types"
)

// UpdateType categorizes the kinds of updates that a datasource can provide.
type UpdateType int

const (
	// UpdateTypeAccount indicates account updates.
	UpdateTypeAccount UpdateType = iota
	// UpdateTypeTransaction indicates transaction updates.
	UpdateTypeTransaction
	// UpdateTypeAccountDeletion indicates account deletion events.
	UpdateTypeAccountDeletion
	// UpdateTypeBlockDetails indicates block details updates.
	UpdateTypeBlockDetails
)

// String returns the string representation of the UpdateType.
func (ut UpdateType) String() string {
	switch ut {
	case UpdateTypeAccount:
		return "AccountUpdate"
	case UpdateTypeTransaction:
		return "Transaction"
	case UpdateTypeAccountDeletion:
		return "AccountDeletion"
	case UpdateTypeBlockDetails:
		return "BlockDetails"
	default:
		return "Unknown"
	}
}

// DatasourceID uniquely identifies a datasource in the pipeline.
// It's used to track the source of data updates and enable filtering.
type DatasourceID struct {
	id string
}

// NewUniqueDatasourceID creates a new datasource ID with a randomly generated unique identifier.
func NewUniqueDatasourceID() DatasourceID {
	return DatasourceID{id: uuid.New().String()}
}

// NewNamedDatasourceID creates a new datasource ID with a specific name.
func NewNamedDatasourceID(name string) DatasourceID {
	return DatasourceID{id: name}
}

// String returns the string representation of the DatasourceID.
func (d DatasourceID) String() string {
	return d.id
}

// Equals checks if two DatasourceIDs are equal.
func (d DatasourceID) Equals(other DatasourceID) bool {
	return d.id == other.id
}

// Update represents a data update in the carbon pipeline.
type Update struct {
	// Type indicates what kind of update this is.
	Type UpdateType

	// Account is set when Type is UpdateTypeAccount.
	Account *AccountUpdate

	// Transaction is set when Type is UpdateTypeTransaction.
	Transaction *TransactionUpdate

	// AccountDeletion is set when Type is UpdateTypeAccountDeletion.
	AccountDeletion *AccountDeletion

	// BlockDetails is set when Type is UpdateTypeBlockDetails.
	BlockDetails *BlockDetails
}

// NewAccountUpdate creates a new Update for an account update.
func NewAccountUpdate(update *AccountUpdate) Update {
	return Update{
		Type:    UpdateTypeAccount,
		Account: update,
	}
}

// NewTransactionUpdate creates a new Update for a transaction update.
func NewTransactionUpdate(update *TransactionUpdate) Update {
	return Update{
		Type:        UpdateTypeTransaction,
		Transaction: update,
	}
}

// NewAccountDeletionUpdate creates a new Update for an account deletion.
func NewAccountDeletionUpdate(deletion *AccountDeletion) Update {
	return Update{
		Type:            UpdateTypeAccountDeletion,
		AccountDeletion: deletion,
	}
}

// NewBlockDetailsUpdate creates a new Update for block details.
func NewBlockDetailsUpdate(details *BlockDetails) Update {
	return Update{
		Type:         UpdateTypeBlockDetails,
		BlockDetails: details,
	}
}

// AccountUpdate represents an update to a Solana account.
type AccountUpdate struct {
	// Pubkey is the public key of the account being updated.
	Pubkey types.Pubkey

	// Account is the new state of the account.
	Account types.Account

	// Slot is the slot number in which this account update was recorded.
	Slot uint64

	// TransactionSignature is the signature of the transaction that caused the update.
	TransactionSignature *types.Signature
}

// TransactionUpdate represents a transaction update in the Solana network.
type TransactionUpdate struct {
	// Signature is the unique signature of the transaction.
	Signature types.Signature

	// Transaction is the versioned transaction data.
	Transaction *solana.Transaction

	// Meta contains metadata about the transaction's status.
	Meta types.TransactionStatusMeta

	// IsVote indicates whether the transaction is a vote transaction.
	IsVote bool

	// Slot is the slot number in which the transaction was recorded.
	Slot uint64

	// Index is the index of the transaction within the slot (block).
	Index *uint64

	// BlockTime is the Unix timestamp of when the transaction was processed.
	BlockTime *int64

	// BlockHash is the block hash that can be used to detect a fork.
	BlockHash *types.Hash
}

// AccountDeletion represents the deletion of a Solana account.
type AccountDeletion struct {
	// Pubkey is the public key of the deleted account.
	Pubkey types.Pubkey

	// Slot is the slot number in which the account was deleted.
	Slot uint64

	// TransactionSignature is the signature of the transaction that caused the deletion.
	TransactionSignature *types.Signature
}

// BlockDetails represents the details of a Solana block.
type BlockDetails struct {
	// Slot is the slot number of this block.
	Slot uint64

	// BlockHash is the hash of the current block.
	BlockHash *types.Hash

	// PreviousBlockHash is the hash of the previous block.
	PreviousBlockHash *types.Hash

	// Rewards contains rewards information associated with the block.
	Rewards []types.Reward

	// NumRewardPartitions is the number of reward partitions in the block.
	NumRewardPartitions *uint64

	// BlockTime is the Unix timestamp indicating when the block was processed.
	BlockTime *int64

	// BlockHeight is the height of the block in the blockchain.
	BlockHeight *uint64
}

// UpdateWithSource pairs an Update with its DatasourceID.
type UpdateWithSource struct {
	Update       Update
	DatasourceID DatasourceID
}

// Datasource defines the interface for data sources that produce updates.
//
// Implementations of this interface are responsible for fetching updates
// and sending them through a channel to be processed by the pipeline.
type Datasource interface {
	// Consume starts consuming updates from the datasource.
	// Updates should be sent to the provided channel along with the datasource ID.
	// The context is used for cancellation.
	// The metrics collection is used for recording performance metrics.
	Consume(
		ctx context.Context,
		id DatasourceID,
		updates chan<- UpdateWithSource,
		metrics *metrics.Collection,
	) error

	// UpdateTypes returns the types of updates this datasource can provide.
	UpdateTypes() []UpdateType
}
