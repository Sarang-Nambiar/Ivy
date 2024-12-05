package CM

type SyncMessage struct {
	Records map[int]Record
	WriteQueue []WriteRequest
}