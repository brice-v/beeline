package db

import (
	"beeline/models"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
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
	d.CreateUser("admin", pw)
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

func (d *DB) CreateUser(username, password string) {
	// Generate "hash" to store from user password
	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// TODO: Properly handle error
		log.Fatal(err)
	}
	d.db.Create(&models.User{
		Username: username,
		Password: string(pwHash),
	})
}

func (d *DB) TotalUserCount() int {
	var count int64
	d.db.Table("users").Count(&count)
	return int(count)
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
		log.Printf("GetPosts error: %s", result.Error.Error())
	}
	return posts
}

func (d *DB) GetSingleUsersPosts(user *models.User) []models.Post {
	var posts []models.Post
	result := d.db.Order("id desc").Find(&posts, "username = ?", user.Username)
	if result.Error != nil {
		log.Printf("GetSingleUsersPosts error: %s", result.Error.Error())
	}
	return posts
}

func (d *DB) GetAllPosts() []models.Post {
	var posts []models.Post
	tx := d.db.Order("id desc").Find(&posts)
	if tx.Error != nil {
		log.Printf("GetAllPosts error: %s", tx.Error)
	}
	return posts
}

func (d *DB) NewPost(p *models.Post) {
	tx := d.db.Create(p)
	if tx.Error != nil {
		log.Printf("NewPost error: %s", tx.Error.Error())
	}
}

func (d *DB) NewPaste(p *models.Paste) {
	tx := d.db.Create(p)
	if tx.Error != nil {
		log.Printf("NewPaste error: %s", tx.Error.Error())
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
		log.Printf("DeleteAuthId error: %s", tx.Error.Error())
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
		log.Printf("IsUserFollowing error: %s", result.Error.Error())
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
		log.Printf("FollowUser error: %s", result.Error.Error())
	}
}

func (d *DB) GetAllPastes(user *models.User) []models.Paste {
	var pastes []models.Paste
	tx := d.db.Where("username = ?", user.Username).Order("id desc").Find(&pastes)
	if tx.Error != nil {
		log.Printf("GetAllPastes error: %s", tx.Error)
	}
	return pastes
}

func (d *DB) GetPaste(user *models.User, id uint64) (models.Paste, bool) {
	var paste models.Paste
	tx := d.db.Where("username = ?", user.Username).Where("id = ?", id).First(&paste)
	if tx.Error != nil {
		log.Printf("GetPaste error: %s, ID: %d, %s", user.String(), id, tx.Error.Error())
		return paste, false
	}
	return paste, true
}
