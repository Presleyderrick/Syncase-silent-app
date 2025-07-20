package db

func GetUserByEmailWithPassword(email string) (User, string, error) {
	var user User
	var hash string
	query := `SELECT id, email, role, password_hash FROM users WHERE email = $1`
	err := DB.QueryRow(query, email).Scan(&user.ID, &user.Email, &user.Role, &hash)
	return user, hash, err
}

func GetAllowedFolders(userID string) ([]string, error) {
	query := `
		SELECT f.path
		FROM user_folders uf
		JOIN folders f ON uf.folder_id = f.id
		WHERE uf.user_id = $1
	`

	rows, err := DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		folders = append(folders, path)
	}
	return folders, nil
}
