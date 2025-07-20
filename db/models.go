package db

type User struct {
	ID    string
	Email string
	Role  string
}

type FolderPermission struct {
	FolderPrefix string
}
