package mail

import (
	"github.com/nicholaskh/pushd/db"
	"gopkg.in/mgo.v2/bson"
	"errors"
	"fmt"
	"reflect"
)
//
//func PushMsgToMail(userIds []string, msgId bson.ObjectId) error {
//	data := bson.M{}
//	data["type"] = MAIL_TYPE_MESSAGE
//	data["data"] = msgId
//	return sendLetterToMail(userIds, &data)
//}
//
//
//func GetUnreadMsgId(userId string) ([]*Letter, error){
//	datas, err := getLettersByType(userId, MAIL_TYPE_MESSAGE)
//	if err != nil {
//		return nil, err
//	}
//
//	letters := make([]*Letter, 0, len(datas))
//	for _, data := range datas {
//		document := data.(bson.M)
//		le := Letter{}
//		le.Id = document["_id"]
//		le.Data = document["data"]
//		le.Data = MAIL_TYPE_MESSAGE
//		letters = append(letters, &le)
//	}
//
//	return letters, nil
//}

func SendLetterToMail(userIds []string, letter *Letter) error {
	bulk := db.MgoSession().DB("pushd").C("mail").Bulk()
	var documents []interface{}

	bs := bson.M{}
	bs["ts"] = letter.Ts
	bs["type"] = letter.Type
	bs["id"] = letter.Id
	bs["data"] = letter.Data

	for _, v := range userIds {
		documents = append(documents, bson.M{"_id": v})
		documents = append(documents, bson.M{"$push": bson.M{"mail": bs}})

	}
	bulk.Upsert(documents...)
	_, err := bulk.Run()
	return err
}



func GetLettersByType(userId string, mailType int) ([]*Letter, error) {
	var result interface{}

	coll := db.MgoSession().DB("pushd").C("mail")
	err := coll.Find(bson.M{"_id": userId}).Select(bson.M{"mail":1}).One(&result)

	if err != nil {
		return nil, err
	}
	letters := make([]*Letter, 0)

	mails := result.(bson.M)["mail"]

	v := reflect.ValueOf(mails)
	if v.Kind() != reflect.Slice {
		return letters, errors.New("GetLettersByType not slice")
	}
	l := v.Len()
	for i := 0; i < l; i++ {
		a := v.Index(i).Interface()
		bs := a.(bson.M)
		letter := Letter{}
		letter.Id = bs["id"].(string)
		letter.Type = bs["type"].(int)
		letter.Ts = bs["ts"].(int64)
		letter.Data = bs["data"].(string)
		letters = append(letters, &letter)
	}

	return letters, err
}

func RemoveLetter(userId string, mailIds []string) error {
	objectIds := make([]bson.ObjectId, 0, len(mailIds))
	for _, id := range mailIds {
		if !bson.IsObjectIdHex(id) {
			return errors.New(fmt.Sprintf("userId:%s is not ObjectId", id))
		}
		objectIds = append(objectIds, bson.ObjectIdHex(id))
	}

	return db.MgoSession().DB("pushd").C("mail").Update(bson.M{"_id": userId},
		bson.M{"$pull":bson.M{"mail":bson.M{"id":bson.M{"$in": mailIds}}}})
}
