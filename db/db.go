package db

import (
	"beeline/models"
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DB struct {
	db *gorm.DB
}

func NewAndMigrate(dbName string) (*DB, error) {
	db, err := gorm.Open(sqlite.Open(dbName))
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&models.User{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&models.Post{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&models.Following{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&models.Auth{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&models.Paste{})
	if err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func (d *DB) CreateAdmin() {
	pw := os.Getenv("BEELINE_ADMIN_PW")
	if pw == "" {
		log.Printf("`BEELINE_ADMIN_PW` is empty")
		return
	}
	d.CreateUser("admin", pw, true)
}

func (d *DB) IncrementFailedLoginAttempts(username string) {
	user, ok := d.FindUser(username)
	if !ok {
		log.Printf("DB::IncrementFailedLoginAttempts: username `%s` not found", username)
		return
	}
	currentFailedLoginAttempts := user.FailedLoginAttempts
	txResult := d.db.Model(&user).Update("failed_login_attempts", currentFailedLoginAttempts+1)
	if txResult.Error != nil {
		log.Printf("DB::IncrementFailedLoginAttempts: error: %s", txResult.Error.Error())
		return
	}
}

func (d *DB) ResetFailedLoginAttempts(username string) {
	user, ok := d.FindUser(username)
	if !ok {
		log.Printf("DB::ResetFailedLoginAttempts: username `%s` not found", username)
		return
	}
	txResult := d.db.Model(&user).Update("failed_login_attempts", 0)
	if txResult.Error != nil {
		log.Printf("DB::ResetFailedLoginAttempts: error: %s", txResult.Error.Error())
		return
	}
}

func (d *DB) FindUser(username string) (*models.User, bool) {
	var user models.User
	txResult := d.db.First(&user, "username = ?", username)
	if txResult.RowsAffected == 0 {
		return nil, false
	}
	return &user, true
}

func (d *DB) CreateUser(username, password string, isAdmin bool) {
	pwHash := generatePasswordHash(password)
	d.db.Create(&models.User{
		Username: username,
		Password: pwHash,
		Admin:    isAdmin,
	})
}

func (d *DB) TotalUserCount() int {
	var count int64
	d.db.Table("users").Count(&count)
	return int(count)
}

func (d *DB) GetAllUsers() []models.User {
	var users []models.User
	tx := d.db.Find(&users)
	if tx.Error != nil {
		log.Printf("DB::GetAllUsers error: %s", tx.Error)
	}
	return users
}

func (d *DB) GetUser(userId uint64) models.User {
	var user models.User
	tx := d.db.Where("id = ?", userId).First(&user)
	if tx.Error != nil {
		log.Printf("DB::GetUser error: %s", tx.Error.Error())
	}
	return user
}

func (d *DB) UpdateUserUsername(userId uint64, username string) {
	tx := d.db.Model(&models.User{}).Where("id = ?", userId).Update("username", username)
	if tx.Error != nil {
		log.Printf("DB::UpdateUserUsername error: %s", tx.Error.Error())
	}
}

func (d *DB) UpdateUserPassword(userId uint64, password string) {
	pwHash := generatePasswordHash(password)
	tx := d.db.Model(&models.User{}).Where("id = ?", userId).Update("password", pwHash)
	if tx.Error != nil {
		log.Printf("DB::UpdateUserPassword error: %s", tx.Error.Error())
	}
}

func (d *DB) UpdateUserAdmin(userId uint64, isAdmin bool) {
	tx := d.db.Model(&models.User{}).Where("id = ?", userId).Update("admin", isAdmin)
	if tx.Error != nil {
		log.Printf("DB::UpdateUserAdmin error: %s", tx.Error.Error())
	}
}

func (d *DB) UpdateUserFailedLoginAttempts(userId uint64, failedLoginAttempts int) {
	tx := d.db.Model(&models.User{}).Where("id = ?", userId).Update("failed_login_attempts", failedLoginAttempts)
	if tx.Error != nil {
		log.Printf("DB::UpdateUserFailedLoginAttempts error: %s", tx.Error.Error())
	}
}

func (d *DB) GetPosts(user *models.User) []models.Post {
	var posts []models.Post
	usersToGetFrom := []string{user.Username}
	var followings []models.Following
	result := d.db.Find(&followings, "follower = ?", user.Username)
	if result.RowsAffected != 0 {
		for _, v := range followings {
			usersToGetFrom = append(usersToGetFrom, v.Username)
		}
	}

	result = d.db.Order("id desc").Find(&posts, "username in (?)", usersToGetFrom)
	if result.Error != nil {
		log.Printf("DB::GetPosts error: %s", result.Error.Error())
	}
	return posts
}

func (d *DB) GetSingleUsersPosts(user *models.User) []models.Post {
	var posts []models.Post
	result := d.db.Order("id desc").Find(&posts, "username = ?", user.Username)
	if result.Error != nil {
		log.Printf("DB::GetSingleUsersPosts error: %s", result.Error.Error())
	}
	return posts
}

func (d *DB) GetAllPosts() []models.Post {
	var posts []models.Post
	tx := d.db.Order("id desc").Find(&posts)
	if tx.Error != nil {
		log.Printf("DB::GetAllPosts error: %s", tx.Error)
	}
	return posts
}

func (d *DB) NewPost(p *models.Post) {
	tx := d.db.Create(p)
	if tx.Error != nil {
		log.Printf("DB::NewPost error: %s", tx.Error.Error())
	}
}

func (d *DB) NewPaste(p *models.Paste) {
	tx := d.db.Create(p)
	if tx.Error != nil {
		log.Printf("DB::NewPaste error: %s", tx.Error.Error())
	}
}

func (d *DB) GetAuthId(un string) (string, bool) {
	var a models.Auth
	result := d.db.First(&a, "username = ?", un)
	if result.Error != nil {
		return "", false
	}
	return a.AuthId, true
}

func (d *DB) SetAuthId(un, authId string) {
	_, ok := d.GetAuthId(un)
	if !ok {
		a := &models.Auth{
			Username: un,
			AuthId:   authId,
		}
		d.db.Create(a)
	} else {
		var a models.Auth
		d.db.First(&a, "username = ?", un)
		a.AuthId = authId
		d.db.Save(&a)
	}
}

func (d *DB) DeleteAuthId(un string) {
	var a models.Auth
	tx := d.db.Where("username = ?", un).Delete(&a)
	if tx.Error != nil {
		log.Printf("DB::DeleteAuthId error: %s", tx.Error.Error())
	}
}

func (d *DB) DeleteAllAuthIds() {
	d.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Auth{})
}

func (d *DB) IsUserFollowing(userToFollow, currentUser string) bool {
	if userToFollow == currentUser {
		return true
	}
	var f models.Following
	result := d.db.First(&f, "username = ? AND follower = ?", userToFollow, currentUser)
	if result.Error != nil {
		log.Printf("DB::IsUserFollowing error: %s", result.Error.Error())
	}
	return result.RowsAffected == 1
}

func (d *DB) FollowUser(userToFollow, currentUser string) {
	if d.IsUserFollowing(userToFollow, currentUser) {
		return
	}
	f := models.Following{
		Username: userToFollow,
		Follower: currentUser,
	}
	result := d.db.Create(&f)
	if result.Error != nil {
		log.Printf("DB::FollowUser error: %s", result.Error.Error())
	}
}

func (d *DB) GetAllPastes(user *models.User) []models.Paste {
	var pastes []models.Paste
	tx := d.db.Where("username = ?", user.Username).Order("id desc").Find(&pastes)
	if tx.Error != nil {
		log.Printf("DB::GetAllPastes error: %s", tx.Error)
	}
	return pastes
}

func (d *DB) GetPaste(user *models.User, id uint64) (models.Paste, bool) {
	var paste models.Paste
	tx := d.db.Where("username = ?", user.Username).Where("id = ?", id).First(&paste)
	if tx.Error != nil {
		log.Printf("DB::GetPaste error: %s, ID: %d, %s", user.String(), id, tx.Error.Error())
		return paste, false
	}
	return paste, true
}
