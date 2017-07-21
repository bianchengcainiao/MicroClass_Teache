package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	//"github.com/hoisie/web"
)

var (
	db *sql.DB
)

func init() {
	initDB()
}

func initDB() {
	//user:password@/dbname
	//forbidden to write "db,err:=sql.Open",otherwise db is a local viriable,can cause panic
	var err error
	db, err = sql.Open("mysql", "root:SmartRoot187*+MiMa@/MicroClass")
	checkErr(err)
	err = db.Ping()
	checkErr(err)
	//_, err = db.Query("create database if not exists MicroClass;")
	//checkErr(err)
	//_, err = db.Query("create table if not exists user_information(userID int primary key auto_increment,userAccount varchar(20),userName varchar(20),password varchar(20),registerTime datetime);")
	//checkErr(err)
}

//login
//if success,return user_information with json data
//else failed,return a string "failed"
func loginServer(c *gin.Context) {
	var (
		userID       string
		userName     string
		userAccount  string
		password     string
		registerTime string
		login        Login
	)

	err := c.BindJSON(&login)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	//get account and password success
	rows, err := db.Query("select userID,userName,userAccount,password,registerTime from user_information where userAccount='" + login.Account + "'and password='" + login.Password + "'")
	checkErr(err)
	defer rows.Close()

	if rows.Next() {
		//found,login success
		err := rows.Scan(&userID, &userName, &userAccount, &password, &registerTime)
		checkErr(err)
		c.JSON(http.StatusOK, gin.H{"userID": userID, "userName": userName, "userAccount": userAccount, "password": password, "registerTime": registerTime})
	} else {
		//not found,login failed
		c.String(http.StatusForbidden, "failed")
	}
}

//register
//if success,return a string "true"
//else if registered before,return a string "registered"
//else return a string "failed"
func registerServer(c *gin.Context) {
	var login Login

	err := c.BindJSON(&login)
	if err != nil {
		c.String(http.StatusOK, "failed")
		return
	}
	//get account and password success
	rows, err := db.Query("select userID,userName,userAccount,password,registerTime from user_information where userAccount='" + login.Account + "'and password='" + login.Password + "'")
	checkErr(err)
	defer rows.Close()

	if rows.Next() {
		//found,regisitered
		c.String(http.StatusOK, "registered")
	} else {
		//not found
		stmt, err := db.Prepare("insert into user_information(userAccount,password,registerTime) values(?,?,now())")
		checkErr(err)
		_, err = stmt.Exec(login.Account, login.Password)
		checkErr(err)
		//c.JSON(http.StatusOK, gin.H{"status": "true"})
		//c.JSON(http.StatusOK, "true")
		c.String(http.StatusOK, "true")
	}
}

func addFriendServer(c *gin.Context) {
	type addFriend struct {
		MyID  string `form:"myID" json:"myID" binding:"required"`
		Phone string `form:"phone" json:"phone" binding:"required"`
	}
	var obj addFriend
	var id int = -1
	err := c.BindJSON(&obj)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	err = db.QueryRow("select userID from user_information where userAccount='" + obj.Phone + "';").Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.String(http.StatusOK, "not exist")
			return
		} else {
			checkErr(err)
		}
	}
	if id != -1 {
		//other man exist
		isFriend, err := db.Query("select listID from friend_information where (user1ID='" + obj.MyID + "' and user2ID='" + strconv.Itoa(id) + "') or (user1ID='" + strconv.Itoa(id) + "' and user2ID='" + obj.MyID + "');")
		checkErr(err)
		if isFriend.Next() {
			//already friend
			c.String(http.StatusOK, "already friend")
		} else {
			//search other man's id
			stmt, err := db.Prepare("insert into friend_information(user1ID,user2ID,applyUser,applyTime) values(?,?,?,now())")
			checkErr(err)
			_, err = stmt.Exec(obj.MyID, id, obj.MyID)
			checkErr(err)
			c.String(http.StatusOK, "true")
			//send add friend request to other man
			//////////////////////////////////////////////////////////////////////////////////////
		}
	} else {
		//not found
		c.String(http.StatusOK, "not exist")
	}
}

func uploadUserImageServer(c *gin.Context) {
	file, header, err := c.Request.FormFile("img")
	filepath := header.Filename
	slice := strings.Split(filepath, "/")
	filename := slice[len(slice)-1]

	_, header2, err := c.Request.FormFile("userID")
	userID := header2.Filename
	//userID := header.Header.Get("userID")

	var beforeImagePath string
	err = db.QueryRow("select imagePath from image where userID=\"" + userID + "\" and imageTag=\"image\";").Scan(&beforeImagePath)
	if err != nil {
		if err == sql.ErrNoRows {
			beforeImagePath = ""
		} else {
			checkErr(err)
		}
	}
	//assure every user has only one image
	stmt, err := db.Prepare("update image set imageTag='before' where userID=?")
	checkErr(err)
	_, err = stmt.Exec(userID)
	checkErr(err)

	stmt, err = db.Prepare("insert into image(imageName,imagePath,localPath,imageTag,userID) values(?,?,?,?,?)")
	checkErr(err)
	_, err = stmt.Exec(filename, "../../res/MicroClass/image/"+userID+"/"+filename, filepath, "image", userID)
	checkErr(err)

	//create file
	err = os.MkdirAll("../../res/MicroClass/image/"+userID, 0770)
	checkErr(err)
	files, _ := ioutil.ReadDir("../../res/MicroClass/image/" + userID)
	if len(files) == 1 {
		_, err = os.Stat("../../res/MicroClass/image/" + userID + "/before")
		if os.IsExist(err) {
			//do nothing
		} else {
			err = os.Mkdir("../../res/MicroClass/image/"+userID+"/before", 0770)
			checkErr(err)
			p := strings.Split(beforeImagePath, "/")
			err = os.Rename("../../res/MicroClass/image/"+userID+"/"+p[len(p)-1], "../../res/MicroClass/image/"+userID+"/before/"+p[len(p)-1])
			checkErr(err)
		}
	} else {
		fmt.Print("files: ")
		fmt.Println(len(files))
	}
	out, err := os.Create("../../res/MicroClass/image/" + userID + "/" + filename)
	checkErr(err)
	defer out.Close()
	//copy data
	_, err = io.Copy(out, file)
	checkErr(err)
	c.String(http.StatusOK, "upload img ok")
}

func uploadVideoServer(c *gin.Context) {
	file, header, err := c.Request.FormFile("video")
	filepath := header.Filename
	slice := strings.Split(filepath, "/")
	filename := slice[len(slice)-1]

	_, header2, err := c.Request.FormFile("userID")
	userID := header2.Filename

	//stmt, err := db.Prepare("insert into video(videoName,videoPath,userID) values(?,?,?)")
	//checkErr(err)
	//_, err = stmt.Exec(filename, filepath, userID)
	//checkErr(err)
	_, err = db.Query("insert into video(videoName,videoPath,localPath,userID) values(\"" + filename + "\",\"" + "../../res/MicroClass/video/" + userID + "/" + filename + "\",\"" + filepath + "\",\"" + userID + "\");")
	checkErr(err)

	//create file
	err = os.MkdirAll("../../res/MicroClass/video/"+userID, 0770)
	checkErr(err)
	out, err := os.Create("../../res/MicroClass/video/" + userID + "/" + filename)
	checkErr(err)
	defer out.Close()
	//copy data
	_, err = io.Copy(out, file)
	checkErr(err)
	c.String(http.StatusOK, "upload video ok")
}

func getCourseServer(c *gin.Context) {

	type course struct {
		CourseID      string `json:"courseID" form:"courseID" binding:"required"`
		CourseName    string `json:"courseName" form:"courseName" binding:"required"`
		CourseTag     string `json:"courseTag" form:"courseTag"`
		Introduction  string `json:"introduction" form:"introduction" binding:"required"`
		TeacherID     string `json:"teacherID" form:"teacherID" binding:"required"`
		CourseImgName string `json:"courseImgName" form:"courseImgName" binding:"required"`
		CourseRequire string `json:"courseRequire" form:"courseRequire" binding:"required"`
	}

	_, header, err := c.Request.FormFile("userID")
	userID := header.Filename

	var courseList []course
	rows, err := db.Query("select courseID,courseName,courseTag,introduction,courseImgName from course where teacherID='" + userID + "';")
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		var c course
		rows.Scan(&c.CourseID, &c.CourseName, &c.CourseTag, &c.Introduction, &c.CourseImgName)
		checkErr(err)
		c.TeacherID = userID
		courseList = append(courseList, c)
	}
	var interfaceSlice []interface{} = make([]interface{}, len(courseList))
	for i, v := range courseList {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}
func getVideosServer(c *gin.Context) {

	type video struct {
		VideoID     int    `json:"videoID" form:"videoID" binding:"required"`
		VideoName   string `json:"videoName" form:"videoName" binding:"required"`
		VideoPath   string `json:"videoPath" form:"videoPath" binding:"required"`
		LocalPath   string `json:"localPath" form:"localPath" binding:"required"`
		VideoSize   int64  `json:"videoSize" form:"videoSize" binding:"required"`
		VideoLength int64  `json:"videoLength" form:"videoLength"`
		VideoTag    string `json:"videoTag" form:"videoTag"`
	}

	_, header, err := c.Request.FormFile("userID")
	userID := header.Filename

	//path := "../../res/MicroClass/video/" + userID
	//_, err = os.Stat(path)
	//if err != nil {
	//	c.JSON(http.StatusNotFound, gin.H{"status": err})
	//	return
	//}
	//files, _ := ioutil.ReadDir(path)
	//var videoList []video
	//for index, file := range files {
	//	if file.IsDir() {
	//		continue
	//	} else {
	//		//v := video{index, file.Name(), "http://" + ServerIP + "/video/" + userID + "/" + file.Name(), file.Size(), 0, ""}
	//		v := video{index, file.Name(), "http://" + ServerIP + "/video/" + userID + "/" + file.Name(), file.Size(), 0, ""}
	//		videoList = append(videoList, v)
	//	}
	//}
	var videoList []video
	rows, err := db.Query("select videoID,videoName,videoPath,localPath from video where userID='" + userID + "';")
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		var v video
		rows.Scan(&v.VideoID, &v.VideoName, &v.VideoPath, &v.LocalPath)
		checkErr(err)
		videoList = append(videoList, v)
	}
	var interfaceSlice []interface{} = make([]interface{}, len(videoList))
	for i, v := range videoList {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}

func videoServer(c *gin.Context) {
	id := c.Param("id")
	name := c.Param("name")

	//buf, err := ioutil.ReadFile("../../res/MicroClass/video/" + id + "/" + name)
	//checkErr(err)

	//c.Data(http.StatusOK, "Multipart/form-data", buf)
	c.File("../../res/MicroClass/video/" + id + "/" + name)
	//c.String

}
func imageServer(c *gin.Context) {
	id := c.Param("id")
	name := c.Param("name")
	if name == "/" {
		var imagePath string
		err := db.QueryRow("select imagePath from image where userID=\"" + id + "\" and imageTag=\"image\";").Scan(&imagePath)
		if err != nil {
			if err == sql.ErrNoRows {
				c.String(http.StatusNotFound, "ok")
				return
			} else {
				checkErr(err)
			}
		}
		checkErr(err)
		vector := strings.Split(imagePath, "/")
		//fmt.Println(imagePath)
		//fmt.Println("../../res/MicroClass/image/" + id + "/" + vector[len(vector)-1])
		c.File("../../res/MicroClass/image/" + id + "/" + vector[len(vector)-1])
	} else {
		c.File("../../res/MicroClass/image/" + id + "/" + name)
	}
}
func getFriendList(c *gin.Context) {
	type user struct {
		UserID        int    `form:"userID" json:"userID" binding:"required"`
		UserName      string `form:"userName" json:"userName"`
		UserImagePath string `form:"userImagePath" json:"userImagePath"`
	}
	var (
		obj     user
		user1ID int
		user2ID int
	)
	err := c.BindJSON(&obj)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	rows, err := db.Query("select user1ID,user2ID from friend_information where (user1ID=" + strconv.Itoa(obj.UserID) + " or user2ID=" + strconv.Itoa(obj.UserID) + ") and isAgree=\"true\";")
	checkErr(err)
	defer rows.Close()

	var userList []user
	for rows.Next() {
		var u user
		rows.Scan(&user1ID, &user2ID)
		checkErr(err)
		if obj.UserID == user1ID {
			u.UserID = user2ID
		} else {
			u.UserID = user1ID
		}
		//using join sentense
		err = db.QueryRow("select userName from user_information where userID=\"" + strconv.Itoa(u.UserID) + "\";").Scan(&u.UserName)
		if err != nil {
			if err == sql.ErrNoRows {
				c.String(http.StatusNotFound, "ok")
				return
			} else {
				checkErr(err)
			}
		}
		str := "select imagePath from image where userID=\"" + strconv.Itoa(u.UserID) + "\" and imageTag=\"image\";"
		err = db.QueryRow(str).Scan(&u.UserImagePath)
		if err != nil {
			if err == sql.ErrNoRows {
				c.String(http.StatusOK, "ok")
				return
			} else {
				checkErr(err)
			}
		} else {
			userList = append(userList, u)
		}
	}

	var interfaceSlice []interface{} = make([]interface{}, len(userList))
	for i, v := range userList {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}

func getQuestionServer(c *gin.Context) {
	type chat struct {
		Message       string `form:"message" json:"message" binding:"required"`
		YouID         string `form:"youID" json:"youID" binding:"required"`
		YouName       string `form:"youName" json:"youName"`
		ChatImagePath string `form:"charImagePath" json:"chatImagePath"`
	}
	type user struct {
		UserID string `form:"userID" json:"userID" binding:"required"`
	}
	var (
		u        user
		ch       chat
		user1ID  string
		user2ID  string
		sendUser string
	)
	err := c.BindJSON(&u)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	//str := "select message,user1ID,user2ID,sendUser from chat_list where (user1ID=\"" + u.UserID + "\" or user2ID=\"" + u.UserID + "\") group by user1ID,user2ID order by sendTime desc limit 1;"
	str := "select message,user1ID,user2ID,sendUser from chat_list where (user1ID=\"" + u.UserID + "\" or user2ID=\"" + u.UserID + "\") group by user1ID,user2ID having user1ID=\"" + u.UserID + "\";"
	fmt.Println(str)
	rows, err := db.Query(str)
	checkErr(err)
	defer rows.Close()

	var list []chat
	for rows.Next() {
		rows.Scan(&ch.Message, &user1ID, &user2ID, &sendUser)
		if u.UserID == user1ID {
			ch.YouID = user2ID
		} else {
			ch.YouID = user1ID
		}
		str := "select userName from user_information where userID=\"" + ch.YouID + "\";"
		fmt.Println(str)
		err = db.QueryRow(str).Scan(&ch.YouName)
		if err != nil {
			if err == sql.ErrNoRows {
				c.String(http.StatusNotFound, "ok1")
				return
			} else {
				checkErr(err)
			}
		}
		str = "select imagePath from image where userID=\"" + ch.YouID + "\" and imageTag=\"image\";"
		fmt.Println(str)
		err = db.QueryRow(str).Scan(&ch.ChatImagePath)
		if err != nil {
			if err == sql.ErrNoRows {
				c.String(http.StatusNotFound, "ok2")
				return
			} else {
				checkErr(err)
			}
		}
		list = append(list, ch)
	}

	var interfaceSlice []interface{} = make([]interface{}, len(list))
	for i, v := range list {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}

func getCourseWithTagServer(c *gin.Context) {
	type course struct {
		CourseID      string `json:"courseID" form:"courseID" binding:"required"`
		CourseName    string `json:"courseName" form:"courseName" binding:"required"`
		CourseTag     string `json:"courseTag" form:"courseTag"`
		Introduction  string `json:"introduction" form:"introduction" binding:"required"`
		TeacherID     string `json:"teacherID" form:"teacherID" binding:"required"`
		CourseImgName string `json:"courseImgName" form:"courseImgName" binding:"required"`
		CourseRequire string `json:"courseRequire" form:"courseRequire" binding:"required"`
	}
	type t struct {
		Tag string `form:"tag" json:"tag" binding:"required"`
	}
	var tt t
	err := c.BindJSON(&tt)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		fmt.Println(err)
		return
	}
	var courseList []course
	str := "select courseID,courseName,courseTag,introduction,teacherID,courseImgName from course where courseTag='" + tt.Tag + "';"
	fmt.Println(tt.Tag)
	fmt.Println(str)
	rows, err := db.Query(str)
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		var c course
		rows.Scan(&c.CourseID, &c.CourseName, &c.CourseTag, &c.Introduction, &c.TeacherID, &c.CourseImgName)
		checkErr(err)
		courseList = append(courseList, c)
	}
	var interfaceSlice []interface{} = make([]interface{}, len(courseList))
	for i, v := range courseList {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}

func getChatWithManServer(c *gin.Context) {
	type chat struct {
		MessageID int    `form:"messageID" json:"messageID" binding:"required"`
		Message   string `form:"message" json:"message" binding:"required"`
		SendID    string `form:"sendID" json:"sendID" binding:"required"`
		RecieveID string `form:"recieveID" json:"recieveID" binding:"required"`
		//ChatImagePath string `form:"charImagePath" json:"chatImagePath"`
	}
	type user struct {
		UserID string `form:"userID" json:"userID" binding:"required"`
		YouID  string `form:"youID" json:"youID" binding:"required"`
	}
	var (
		u        user
		ch       chat
		user1ID  string
		user2ID  string
		sendUser string
	)
	err := c.BindJSON(&u)
	if err != nil {
		fmt.Println(u.UserID)
		fmt.Println(err)
		c.String(http.StatusBadRequest, "failed")
		return
	}
	rows, err := db.Query("select listID,message,user1ID,user2ID,sendUser from chat_list where ((user1ID=\"" + u.UserID + "\" and user2ID=\"" + u.YouID + "\") or (user1ID=\"" + u.YouID + "\" and user2ID=\"" + u.UserID + "\"));")
	checkErr(err)
	defer rows.Close()

	var list []chat
	for rows.Next() {
		rows.Scan(&ch.MessageID, &ch.Message, &user1ID, &user2ID, &sendUser)
		if u.UserID == sendUser {
			if u.UserID == user1ID {
				ch.SendID = user1ID
				ch.RecieveID = user2ID
			} else {
				ch.SendID = user2ID
				ch.RecieveID = user1ID
			}
		} else {
			if u.UserID == user1ID {
				ch.SendID = user2ID
				ch.RecieveID = user1ID
			} else {
				ch.SendID = user1ID
				ch.RecieveID = user2ID
			}
		}
		//str := "select imagePath from image where userID=\"" + ch.YouID + "\" and imageTag=\"image\";"
		//fmt.Println(str)
		//err = db.QueryRow(str).Scan(&ch.ChatImagePath)
		//checkErr(err)
		list = append(list, ch)
	}

	var interfaceSlice []interface{} = make([]interface{}, len(list))
	for i, v := range list {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}
func getCommentsServer(c *gin.Context) {
	type comment struct {
		CommentID    string `form:"commentID" json:"commentID" binding:"required"`
		UserID       string `form:"userID" json:"userID" binding:"required"`
		UserName     string `form:"userName" json:"userName" binding:"required"`
		ImagePath    string `form:"imagePath" json:"imagePath" binding:"required"`
		YouCommentID string `form:"youCommentID" json:"youCommentID" binding:"required"`
		VideoID      string `form:"videoID" json:"videoID" binding:"required"`
		Msg          string `form:"msg" json:"msg" binding:"required"`
		Star         int    `form:"star" json:"star" binding:"required"`
		PublishTime  string `form:"publishTime" json:"publishTime" binding:"required"`
	}
	type video struct {
		VideoID string `form:"videoID" json:"videoID" binding:"required"`
	}
	var (
		co comment
		v  video
	)
	err := c.BindJSON(&v)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	rows, err := db.Query("select commentID,userID,YouCommentID,videoID,msg,star,publishTime from comment_list where (videoID=\"" + v.VideoID + "\" and youCommentID =\"-1\") order by star desc;")
	checkErr(err)
	defer rows.Close()

	var list []comment
	for rows.Next() {
		rows.Scan(&co.CommentID, &co.UserID, &co.YouCommentID, &co.VideoID, &co.Msg, &co.Star, &co.PublishTime)
		str := "select imagePath from image where userID=\"" + co.UserID + "\" and imageTag=\"image\";"
		err = db.QueryRow(str).Scan(&co.ImagePath)
		if err != nil {
			if err == sql.ErrNoRows {
				c.String(http.StatusNotFound, "ok")
				return
			} else {
				checkErr(err)
			}
		}
		err = db.QueryRow("select userName from user_information where userID=\"" + co.UserID + "\";").Scan(&co.UserName)
		if err != nil {
			if err == sql.ErrNoRows {
				c.String(http.StatusNotFound, "ok")
				return
			} else {
				checkErr(err)
			}
		}
		list = append(list, co)
	}

	var interfaceSlice []interface{} = make([]interface{}, len(list))
	for i, v := range list {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}
func updateCommentStarServer(c *gin.Context) {
	type update struct {
		CommentID  string `form:"commentID" json:"commentID" binding:"required"`
		LatestStar string `form:"star" json:"star" binding:"required"`
	}
	var u update
	err := c.BindJSON(&u)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	stmt, err := db.Prepare("update comment_list set star=? where commentID=?;")
	checkErr(err)
	star, err := strconv.Atoi(u.LatestStar)
	checkErr(err)
	id, err := strconv.Atoi(u.CommentID)
	checkErr(err)
	_, err = stmt.Exec(star, id)
	checkErr(err)
	c.JSON(http.StatusOK, "true")
}
func createCommentServer(c *gin.Context) {
	type comment struct {
		CommentID    string `form:"commentID" json:"commentID"`
		UserID       string `form:"userID" json:"userID" binding:"required"`
		UserName     string `form:"userName" json:"userName"`
		ImagePath    string `form:"imagePath" json:"imagePath"`
		YouCommentID string `form:"youCommentID" json:"youCommentID" binding:"required"`
		VideoID      string `form:"videoID" json:"videoID" binding:"required"`
		Msg          string `form:"msg" json:"msg" binding:"required"`
		Star         string `form:"star" json:"star"`
		PublishTime  string `form:"publishTime" json:"publishTime"`
	}
	var co comment
	err := c.BindJSON(&co)
	if err != nil {
		fmt.Println(err)
		c.String(http.StatusBadRequest, "failed")
		return
	}
	stmt, err := db.Prepare("insert into comment_list(userID,youCommentID,videoID,msg,star,publishTime) values(?,?,?,?,?,now())")
	checkErr(err)
	_, err = stmt.Exec(co.UserID, co.YouCommentID, co.VideoID, co.Msg /*co.Star*/, 0)
	checkErr(err)
	c.JSON(http.StatusOK, "true")
}
func getCommentAnswerServer(c *gin.Context) {
	type user struct {
		YouCommentID string `form:"youCommentID" json:"youCommentID" binding:"required"`
		VideoID      string `form:"videoID" json:"videoID" binding:"required"`
	}
	type answer struct {
		UserID    string `form:"userID" json:"userID" binding:"required"`
		UserName  string `form:"userName" json:"userName" binding:"required"`
		ImagePath string `form:"imagePath" json:"imagePath" binding:"required"`
		Msg       string `form:"msg" json:"msg" binding:"required"`
	}
	var u user
	var a answer
	err := c.BindJSON(&u)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	err = db.QueryRow("select userID,msg from comment_list where (youCommentID="+u.YouCommentID+" and videoID="+u.VideoID+");").Scan(&a.UserID, &a.Msg)
	if err != nil {
		if err == sql.ErrNoRows {
			c.String(http.StatusNotFound, "ok")
			return
		} else {
			checkErr(err)
		}
	}
	err = db.QueryRow("select userName from user_information where (userID=" + a.UserID + ");").Scan(&a.UserName)
	if err != nil {
		if err == sql.ErrNoRows {
			c.String(http.StatusNotFound, "ok")
			return
		} else {
			checkErr(err)
		}
	}
	err = db.QueryRow("select imagePath from image where (userID=" + a.UserID + " and imageTag=\"image\");").Scan(&a.ImagePath)
	if err != nil {
		if err == sql.ErrNoRows {
			c.String(http.StatusNotFound, "true")
			return
		} else {
			checkErr(err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"userID": a.UserID, "userName": a.UserName, "imagePath": a.ImagePath, "msg": a.Msg})
}
func SendMessageServer(c *gin.Context) {
	type msg struct {
		Message   string `form:"message" json:"message" binding:"required"`
		SendID    string `form:"sendID" json:"sendID" binding:"required"`
		RecieveID string `form:"recieveID" json:"recieveID" binding:"required"`
	}
	var m msg
	err := c.BindJSON(&m)
	if err != nil {
		fmt.Println(err)
		c.String(http.StatusBadRequest, "failed")
		return
	}
	stmt, err := db.Prepare("insert into chat_list(message,user1ID,user2ID,sendUser,sendTime) values(?,?,?,?,now())")
	checkErr(err)
	_, err = stmt.Exec(m.Message, m.SendID, m.RecieveID, m.SendID)
	checkErr(err)
	c.JSON(http.StatusOK, "true")
}
func getHomeworkServer(c *gin.Context) {
	type homework struct {
		HomeworkID   string   `form:"homeworkID" json:"homeworkID" binding:"required"`
		Content      string   `form:"content" json:"content" binding:"required"`
		CourseID     string   `form:"courseID" json:"courseID" binding:"required"`
		Option       []string `form:"option" json:"option" binding:"required"`
		RealAnswer   string   `form:"realAnswer" json:"realAnswer" binding:"required"`
		HomeworkType string   `form:"homeworkType" json:"homeworkType" binding:"required"`
	}
	type course struct {
		CourseID string `form:"courseID" json:"courseID" binding:"required"`
	}
	var (
		h  homework
		co course
	)
	fmt.Println("aaaaaaaaaaaaaaa")
	err := c.BindJSON(&co)
	if err != nil {
		fmt.Println("bbbbbbbbbbbbbbb")
		c.String(http.StatusBadRequest, "failed")
		return
	}
	fmt.Println("ccccccccccccccc")
	str := "select homeworkID,content,realAnswer,homeworkType from homework_list where courseID=\"" + co.CourseID + "\" order by HomeworkID;"
	fmt.Println(str)
	rows, err := db.Query(str)
	checkErr(err)
	defer rows.Close()

	var list []homework
	for rows.Next() {
		rows.Scan(&h.HomeworkID, &h.Content, &h.RealAnswer, &h.HomeworkType)
		str := "select content from options where homeworkID=\"" + h.HomeworkID + "\" and courseID=\"" + co.CourseID + "\" order by optionID asc;"
		rows1, err := db.Query(str)
		checkErr(err)
		defer rows1.Close()
		var op string
		for rows1.Next() {
			rows1.Scan(&op)
			h.Option = append(h.Option, op)
		}
		h.CourseID = co.CourseID
		list = append(list, h)
		h.Option = nil
	}

	var interfaceSlice []interface{} = make([]interface{}, len(list))
	for i, v := range list {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}
func getCourseMenus(c *gin.Context) {
	type menu struct {
		MenuID    string `form:"menuID" json:"menuID" binding:"required"`
		Content   string `form:"content" json:"content" binding:"required"`
		VideoID   string `form:"videoID" json:"videoID" binding:"required"`
		VideoName string `form:"videoName" json:"videoName" binding:"required"`
		VideoPath string `form:"videoPath" json:"videoPath" binding:"required"`
		LocalPath string `form:"localPath" json:"localPath" binding:"required"`
	}
	type course struct {
		CourseID string `form:"courseID" json:"courseID" binding:"required"`
	}
	var (
		m  menu
		co course
	)
	err := c.BindJSON(&co)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	str := "select menuID,content,course_menu_list.videoID,videoName,videoPath,localPath from course_menu_list,video where video.videoID=course_menu_list.videoID and courseID=\"" + co.CourseID + "\" order by menuID;"
	fmt.Println("str: " + str)
	rows, err := db.Query(str)
	checkErr(err)
	defer rows.Close()

	var list []menu
	for rows.Next() {
		rows.Scan(&m.MenuID, &m.Content, &m.VideoID, &m.VideoName, &m.VideoPath, &m.LocalPath)
		fmt.Println(m)
		list = append(list, m)
	}

	var interfaceSlice []interface{} = make([]interface{}, len(list))
	for i, v := range list {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}
func getVideoMenus(c *gin.Context) {
	type menu struct {
		MenuID      string `form:"menuID" json:"menuID" binding:"required"`
		Content     string `form:"content" json:"content" binding:"required"`
		CurrentTime string `form:"currentTime" json:"currentTime" binding:"required"`
	}
	type video struct {
		VideoID string `form:"videoID" json:"videoID" binding:"required"`
	}
	var (
		m menu
		v video
	)
	err := c.BindJSON(&v)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	str := "select menuID,content,currentTime from menu_list where videoID=\"" + v.VideoID + "\" order by menuID;"
	fmt.Println("str: " + str)
	rows, err := db.Query(str)
	checkErr(err)
	defer rows.Close()

	var list []menu
	for rows.Next() {
		rows.Scan(&m.MenuID, &m.Content, &m.CurrentTime)
		fmt.Println(m)
		list = append(list, m)
	}

	var interfaceSlice []interface{} = make([]interface{}, len(list))
	for i, v := range list {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}
func getClassOfCourse(c *gin.Context) {
	type Teacher struct {
		UserID   string `form:"userID" json:"userID" binding:"required"`
		UserName string `form:"userName" json:"userName" binding:"required"`
	}
	type class struct {
		ClassID      string    `form:"classID" json:"classID" binding:"required"`
		ClassName    string    `form:"className" json:"className" binding:"required"`
		ClassMemeber []Teacher `form:"classMemeber" json:"classMember" binding:"required"`
	}
	type course struct {
		CourseID string `form:"courseID" json:"courseID" binding:"required"`
	}
	var (
		cl class
		co course
		t  Teacher
	)
	err := c.BindJSON(&co)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	str := "select classID from goToClass where courseID=\"" + co.CourseID + "\" ;"
	//fmt.Println("str: " + str)
	rows, err := db.Query(str)
	checkErr(err)
	defer rows.Close()

	var list []class
	for rows.Next() {
		rows.Scan(&cl.ClassID)
		err = db.QueryRow("select className from class where classID=\"" + cl.ClassID + "\";").Scan(&cl.ClassName)
		if err != nil {
			if err == sql.ErrNoRows {
				c.String(http.StatusNotFound, "true")
				return
			} else {
				checkErr(err)
			}
		}
		rows1, err := db.Query("select userID,userName from user_information where classID=\"" + cl.ClassID + "\";")
		checkErr(err)
		defer rows1.Close()
		for rows1.Next() {
			rows1.Scan(&t.UserID, &t.UserName)
			cl.ClassMemeber = append(cl.ClassMemeber, t)
		}
		list = append(list, cl)
	}

	var interfaceSlice []interface{} = make([]interface{}, len(list))
	for i, v := range list {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}
func getTeacherOfCourse(c *gin.Context) {
	type Teacher struct {
		UserID   string `form:"userID" json:"userID" binding:"required"`
		UserName string `form:"userName" json:"userName" binding:"required"`
	}
	type course struct {
		CourseID string `form:"courseID" json:"courseID" binding:"required"`
	}
	var (
		co course
		t  Teacher
	)
	err := c.BindJSON(&co)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	str := "select teacherID from course where courseID=\"" + co.CourseID + "\" ;"
	//fmt.Println("str: " + str)
	rows, err := db.Query(str)
	checkErr(err)
	defer rows.Close()

	var list []Teacher
	for rows.Next() {
		rows.Scan(&t.UserID)
		err = db.QueryRow("select userName from user_information where userID=\"" + t.UserID + "\";").Scan(&t.UserName)
		if err != nil {
			if err == sql.ErrNoRows {
				c.String(http.StatusNotFound, "true")
				return
			} else {
				checkErr(err)
			}
		}
		list = append(list, t)
	}

	var interfaceSlice []interface{} = make([]interface{}, len(list))
	for i, v := range list {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}
func getClassList(c *gin.Context) {
	type teacher struct {
		UserID    string `form:"userID" json:"userID" binding:"required"`
		UserName  string `form:"userName" json:"userName" binding:"required"`
		ImageName string `form:"imageName" json:"imageName" binding:"required"`
	}
	type class struct {
		ClassID string `form:"classID" json:"classID" binding:"required"`
	}
	var (
		t  teacher
		cl class
	)
	err := c.BindJSON(&cl)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	str := "select user_information.userID,userName,imageName from user_information,image where user_information.userID=image.userID and imageTag=\"image\" and classID=" + cl.ClassID + ";"
	fmt.Println("str: " + str)
	rows, err := db.Query(str)
	checkErr(err)
	defer rows.Close()

	var list []teacher
	for rows.Next() {
		rows.Scan(&t.UserID, &t.UserName, &t.ImageName)
		fmt.Println(t)
		list = append(list, t)
	}

	var interfaceSlice []interface{} = make([]interface{}, len(list))
	for i, v := range list {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}

func addCourseMenu(c *gin.Context) {
	type courseMenu struct {
		UserID   string `form:"userID" json:"userID" binding:"required"`
		CourseID string `form:"courseID" json:"courseID" binding:"required"`
		Content  string `form:"content" json:"content" binding:"required"`
	}
	file, header, err := c.Request.FormFile("video")
	filepath := header.Filename
	slice := strings.Split(filepath, "/")
	filename := slice[len(slice)-1]

	tmp, _, err := c.Request.FormFile("info")
	defer tmp.Close()
	if err != nil {
		c.String(http.StatusInternalServerError, "123")
	}
	info := bytes.NewBuffer(nil)
	if _, err := io.Copy(info, tmp); err != nil {
		c.String(http.StatusInternalServerError, "456")
	}

	var cm courseMenu
	err = json.Unmarshal(info.Bytes(), &cm)

	_, err = db.Query("insert into video(videoName,videoPath,localPath,userID) values(\"" + filename + "\",\"" + "../../res/MicroClass/video/" + cm.UserID + "/" + filename + "\",\"" + filepath + "\",\"" + cm.UserID + "\");")
	checkErr(err)

	//create file
	err = os.MkdirAll("../../res/MicroClass/video/"+cm.UserID, 0770)
	checkErr(err)
	out, err := os.Create("../../res/MicroClass/video/" + cm.UserID + "/" + filename)
	checkErr(err)
	defer out.Close()
	//copy data
	_, err = io.Copy(out, file)
	checkErr(err)

	var videoID int
	err = db.QueryRow("select videoID from video where videoName=\"" + filename + "\" and userID=\"" + cm.UserID + "\";").Scan(&videoID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.String(http.StatusInternalServerError, "false")
			return
		} else {
			checkErr(err)
		}
	}
	_, err = db.Query("insert into course_menu_list(menuID,content,videoID,courseID) values(1,\"" + cm.Content + "\",\"" + strconv.Itoa(videoID) + "\",\"" + cm.CourseID + "\");")
	checkErr(err)

	c.String(http.StatusOK, "upload video ok")
}
func applyCourse(c *gin.Context) {
	type course struct {
		UserID        string `form:"userID" json:"userID" binding:"required"`
		CourseID      string `form:"courseID" json:"courseID" binding:"required"`
		CourseName    string `form:"courseName" json:"courseName" binding:"required"`
		CourseTag     string `form:"courseTag" json:"courseTag" binding:"required"`
		CourseRequire string `form:"courseRequire" json:"courseRequire" binding:"required"`
	}
	file, header, err := c.Request.FormFile("img")
	filepath := header.Filename
	slice := strings.Split(filepath, "/")
	filename := slice[len(slice)-1]

	tmp, _, err := c.Request.FormFile("info")
	defer tmp.Close()
	if err != nil {
		c.String(http.StatusInternalServerError, "123")
	}
	info := bytes.NewBuffer(nil)
	if _, err := io.Copy(info, tmp); err != nil {
		c.String(http.StatusInternalServerError, "456")
	}

	var co course
	err = json.Unmarshal(info.Bytes(), &co)

	//fmt.Println("fileName: " + filename)
	//fmt.Println("filePath: " + filepath)
	//fmt.Println("courseName: " + co.CourseName)
	//fmt.Println("courseTag: " + co.CourseTag)
	//fmt.Println("courseRequire: " + co.CourseRequire)

	stmt, err := db.Prepare("insert into course(courseName,courseTag,introduction,teacherID,courseImgName) values(?,?,?,?,?)")
	checkErr(err)
	_, err = stmt.Exec(co.CourseName, co.CourseTag, co.CourseRequire, co.UserID, filename)
	checkErr(err)

	//create file
	err = os.MkdirAll("../../res/MicroClass/image/"+co.UserID, 0770)
	checkErr(err)
	out, err := os.Create("../../res/MicroClass/image/" + co.UserID + "/" + filename)
	checkErr(err)
	defer out.Close()
	//copy data
	_, err = io.Copy(out, file)
	checkErr(err)

	err = db.QueryRow("select courseID from course where courseName=\"" + co.CourseName + "\" and introduction=\"" + co.CourseRequire + "\" and courseTag=\"" + co.CourseTag + "\" and teacherID=\"" + co.UserID + "\" and courseImgName=\"" + filename + "\";").Scan(&co.CourseID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.String(http.StatusNotFound, "true")
			return
		} else {
			checkErr(err)
		}
	}
	checkErr(err)
	c.JSON(http.StatusOK, gin.H{"courseID": co.CourseID})
}
func getClass(c *gin.Context) {
	type class struct {
		ClassID   string `form:"classID" json:"classID" binding:"required"`
		ClassName string `form:"className" json:"className" binding:"required"`
	}
	str := "select classID,className from class;"
	rows, err := db.Query(str)
	checkErr(err)
	defer rows.Close()

	var cl class
	var list []class
	for rows.Next() {
		rows.Scan(&cl.ClassID, &cl.ClassName)
		list = append(list, cl)
	}

	var interfaceSlice []interface{} = make([]interface{}, len(list))
	for i, v := range list {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}
func addClass(c *gin.Context) {
	type class struct {
		ClassID  string `form:"classID" json:"classID" binding:"required"`
		CourseID string `form:"courseID" json:"courseID" binding:"required"`
	}
	var cl class
	err := c.BindJSON(&cl)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	str := "insert into goToClass(courseID,classID) values(\"" + cl.CourseID + "\",\"" + cl.ClassID + "\");"
	fmt.Println("str: " + str)
	rows, err := db.Query(str)
	checkErr(err)
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"status": "true"})
}
func searchFriendApply(c *gin.Context) {
	type you struct {
		ListID   string `form:"listID" json:"listID" binding:"required"`
		UserID   string `form:"userID" json:"userID" binding:"required"`
		UserName string `form:"userName" json:"userName" binding:"required"`
	}
	type user struct {
		UserID string `form:"userID" json:"userID" binding:"required"`
	}
	var u user
	err := c.BindJSON(&u)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	//str := "select friend_information.listID,userID,userName from user_information,friend_information where (user1ID=" + u.UserID + " or user2ID=" + u.UserID + ") and applyUser != " + u.UserID + " and isAgree=\"false\" and (user1ID=userID or user2ID=userID);"
	str := "select listID,user1ID,user2ID from friend_information where (user1ID=" + u.UserID + " or user2ID=" + u.UserID + ") and applyUser!=" + u.UserID + " and isAgree=\"false\";"
	rows, err := db.Query(str)
	checkErr(err)
	defer rows.Close()

	var y you
	var list []you
	var user1ID, user2ID string
	for rows.Next() {
		rows.Scan(&y.ListID, &user1ID, &user2ID)
		if user1ID == u.UserID {
			y.UserID = user2ID
		} else {
			y.UserID = user1ID
		}
		str = "select userName from user_information where userID=" + y.UserID + ";"
		rows1, err := db.Query(str)
		checkErr(err)
		defer rows1.Close()
		for rows1.Next() {
			rows1.Scan(&y.UserName)
		}
		list = append(list, y)
	}

	var interfaceSlice []interface{} = make([]interface{}, len(list))
	for i, v := range list {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}
func agreeApply(c *gin.Context) {
	type apply struct {
		ListID string `form:"listID" json:"listID" binding:"required"`
	}
	var a apply
	err := c.BindJSON(&a)
	if err != nil {
		c.String(http.StatusBadRequest, "failed")
		return
	}
	str := "update friend_information set isAgree=\"true\" where listID=" + a.ListID + ";"
	rows, err := db.Query(str)
	checkErr(err)
	defer rows.Close()

	c.JSON(http.StatusOK, "true")
}
func getGroupMsg(c *gin.Context) {
	type chat struct {
		MessageID   int    `form:"messageID" json:"messageID" binding:"required"`
		Message     string `form:"message" json:"message" binding:"required"`
		SendID      string `form:"sendID" json:"sendID" binding:"required"`
		SendName    string `form:"sendName" json:"sendName" binding:"required"`
		CourseID    string `form:"recieveID" json:"recieveID" binding:"required"`
		IsGroupChat bool   `form:"isGroupChat" json:"isGroupChat" binding:"required"`
	}
	type user struct {
		UserID   string `form:"userID" json:"userID" binding:"required"`
		CourseID string `form:"courseID" json:"courseID" binding:"required"`
	}
	var (
		u  user
		ch chat
	)
	err := c.BindJSON(&u)
	if err != nil {
		fmt.Println(u.UserID)
		fmt.Println(err)
		c.String(http.StatusBadRequest, "failed")
		return
	}
	rows, err := db.Query("select listID,message,sendUser from group_msg where courseID=" + u.CourseID + ";")
	checkErr(err)
	defer rows.Close()

	var list []chat
	for rows.Next() {
		rows.Scan(&ch.MessageID, &ch.Message, &ch.SendID)
		str := "select userName from user_information where userID=\"" + ch.SendID + "\";"
		fmt.Println(str)
		err = db.QueryRow(str).Scan(&ch.SendName)
		checkErr(err)
		ch.IsGroupChat = true
		ch.CourseID = u.CourseID
		list = append(list, ch)
	}

	var interfaceSlice []interface{} = make([]interface{}, len(list))
	for i, v := range list {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}
func SendGroupMessage(c *gin.Context) {
	type msg struct {
		Message  string `form:"message" json:"message" binding:"required"`
		SendID   string `form:"sendID" json:"sendID" binding:"required"`
		CourseID string `form:"courseID" json:"courseID" binding:"required"`
	}
	var m msg
	err := c.BindJSON(&m)
	if err != nil {
		fmt.Println(err)
		c.String(http.StatusBadRequest, "failed")
		return
	}
	stmt, err := db.Prepare("insert into group_msg(message,sendUser,sendTime,courseID) values(?,?,now(),?)")
	checkErr(err)
	_, err = stmt.Exec(m.Message, m.SendID, m.CourseID)
	checkErr(err)
	c.JSON(http.StatusOK, "true")
}
func NewClass(c *gin.Context) {
	type class struct {
		ClassName string `form:"className" json:"className" binding:"required"`
	}
	var cl class
	err := c.BindJSON(&cl)
	if err != nil {
		fmt.Println(err)
		c.String(http.StatusBadRequest, "failed")
		return
	}
	stmt, err := db.Prepare("insert into class(className) values(?)")
	checkErr(err)
	_, err = stmt.Exec(cl.ClassName)
	checkErr(err)

	c.String(http.StatusOK, "ok")
}
func AddStudent(c *gin.Context) {
	type class struct {
		ClassID    string `form:"classID" json:"classID" binding:"required"`
		UserID     string `form:"userID" json:"userID" binding:"required"`
		IsSelected string `form:"isSelected" json:"isSelected" binding:"required"`
	}
	var cl class
	err := c.BindJSON(&cl)
	if err != nil {
		fmt.Println(err)
		c.String(http.StatusBadRequest, "failed")
		return
	}
	var str string
	if cl.IsSelected == "true" {
		str = "update user_information set classID=" + cl.ClassID + " where userID=" + cl.UserID + ";"
		rows, err := db.Query(str)
		checkErr(err)
		defer rows.Close()
	} else {
		//str = "update user_information set classID=-1 where userID=" + cl.UserID + ";"
	}
	c.JSON(http.StatusOK, gin.H{"status": "true"})
}
func GetStudent(c *gin.Context) {
	type student struct {
		UserID   string `form:"userID" json:"userID" binding:"required"`
		UserName string `form:"userName" json:"userName" binding:"required"`
		ClassID  string `form:"classID" json:"classID" binding:"required"`
	}
	str := "select userID,userName,classID from user_information;"
	rows, err := db.Query(str)
	checkErr(err)
	defer rows.Close()

	var s student
	var list []student
	for rows.Next() {
		rows.Scan(&s.UserID, &s.UserName, &s.ClassID)
		list = append(list, s)
	}

	var interfaceSlice []interface{} = make([]interface{}, len(list))
	for i, v := range list {
		interfaceSlice[i] = v
	}
	c.JSON(http.StatusOK, interfaceSlice)
}

func main() {
	defer db.Close()

	//register a default router
	router := gin.Default()

	router.POST("/login", loginServer)
	router.POST("/register", registerServer)
	router.POST("/add_friend", addFriendServer)
	router.POST("/upload_user_image", uploadUserImageServer)
	router.POST("/upload_video", uploadVideoServer)
	router.POST("/get_videos", getVideosServer)
	router.GET("/video/:id/:name", videoServer)
	router.GET("/image/:id/*name", imageServer)
	router.POST("/get_friend_list", getFriendList)
	router.POST("/get_question", getQuestionServer)
	router.POST("/get_course_with_tag", getCourseWithTagServer)
	router.POST("/get_chat_with_man", getChatWithManServer)
	router.POST("/get_comments", getCommentsServer)
	router.POST("/update_comment_star", updateCommentStarServer)
	router.POST("/create_comment", createCommentServer)
	router.POST("/get_comment_answer", getCommentAnswerServer)
	router.POST("/send_message", SendMessageServer)
	router.POST("/get_homeworks", getHomeworkServer)
	router.POST("/get_videoMenu", getVideoMenus)
	router.POST("/get_courses", getCourseServer)
	router.POST("/get_courseMenu", getCourseMenus)
	router.POST("/getClassOfCourse", getClassOfCourse)
	router.POST("/getTeacherOfCourse", getTeacherOfCourse)
	router.POST("/get_classlist", getClassList)
	router.POST("/apply_course", applyCourse)
	router.POST("/add_course_menu", addCourseMenu)
	router.POST("/get_class", getClass)
	router.POST("/add_class", addClass)
	router.POST("/search_friend_apply", searchFriendApply)
	router.POST("/agree_apply", agreeApply)
	router.POST("/get_group_msg", getGroupMsg)
	router.POST("/new_class", NewClass)
	router.POST("/send_group_message", SendGroupMessage)
	router.POST("/add_student", AddStudent)
	router.POST("/get_student", GetStudent)

	router.Run(":8080")
}

func checkErr(err error) {
	if err != nil {
		//log.Fatal(err)
		panic(err.Error())
	}
}
