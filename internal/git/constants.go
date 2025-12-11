package git

// Batch sizes for database processing to ensure scalability
const (
	FileBatchSize        = 1000 // Number of file inventory records to upsert at once
	CommitBatchSize      = 100  // Number of commit records to process before flushing to DB
	ContributorBatchSize = 1000 // Number of contributor records to upsert at once

	// Buffer sizes for file reading
	ScannerInitialBufferSize = 64 * 1024   // 64KB initial buffer
	ScannerMaxBufferSize     = 1024 * 1024 // 1MB max buffer for long lines
)
