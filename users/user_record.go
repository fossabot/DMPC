package users

import (
	"crypto/rsa"
	"time"
)

/*
	Record of a user
	Keeps track of granual timestamps for changes
*/
type keyRecord struct {
	Key 		rsa.PublicKey
	UpdatedAt 	time.Time
}
type booleanRecord struct {
	Ok 			bool
	UpdatedAt 	time.Time
}

type permissionsRecord struct {
	Channel 	channelPermissionsRecord
	User 		userPermissionsRecord
	UpdatedAt 	time.Time
}

type channelPermissionsRecord struct {
	Add 		booleanRecord
	UpdatedAt 	time.Time
}

type userPermissionsRecord struct {
	Add 				booleanRecord
	Remove 				booleanRecord
	EncKeyUpdate 		booleanRecord
	SignKeyUpdate 		booleanRecord
	PermissionsUpdate 	booleanRecord
	UpdatedAt 			time.Time
}

type userRecord struct {
	Id 			string
	EncKey 		keyRecord
	SignKey 	keyRecord
	Permissions permissionsRecord
	Active 		booleanRecord
	CreatedAt 	time.Time
	UpdatedAt 	time.Time
}


func (record *userRecord) applyUpdateRequest(req *UserRequest) {
	for _,field := range req.FieldsUpdated {
		switch field {
			case "active":
				if(record.Active.update(req.Data.Active, req.Timestamp)) {
					record.UpdatedAt = req.Timestamp
				}
			case "encKey":
				if(record.EncKey.update(*req.Data.encKeyObject, req.Timestamp)) {
					record.UpdatedAt = req.Timestamp
				}
			case "signKey":
				if(record.SignKey.update(*req.Data.signKeyObject, req.Timestamp)) {
					record.UpdatedAt = req.Timestamp
				}
			case "permissions.channel.add":
				if record.Permissions.Channel.Add.update(req.Data.Permissions.Channel.Add, req.Timestamp) {
					record.UpdatedAt = req.Timestamp
					record.Permissions.UpdatedAt = req.Timestamp
					record.Permissions.Channel.UpdatedAt = req.Timestamp
				}

			case "permissions.user.add", "permissions.user.remove", "permissions.user.encKeyUpdate", "permissions.user.signKeyUpdate", "permissions.user.permissionsUpdate":
				var perm *booleanRecord
				var reqVal bool
				switch(field) {
					case "permissions.user.add":
						perm = &record.Permissions.User.Add
						reqVal = req.Data.Permissions.User.Add
					case "permissions.user.remove":
						perm = &record.Permissions.User.Remove
						reqVal = req.Data.Permissions.User.Remove
					case "permissions.user.encKeyUpdate":
						perm = &record.Permissions.User.EncKeyUpdate
						reqVal = req.Data.Permissions.User.EncKeyUpdate
					case "permissions.user.signKeyUpdate":
						perm = &record.Permissions.User.SignKeyUpdate
						reqVal = req.Data.Permissions.User.SignKeyUpdate
					case "permissions.user.permissionsUpdate":
						perm = &record.Permissions.User.PermissionsUpdate
						reqVal = req.Data.Permissions.User.PermissionsUpdate
				}

				if perm.update(reqVal, req.Timestamp) {
					record.UpdatedAt = req.Timestamp
					record.Permissions.UpdatedAt = req.Timestamp
					record.Permissions.User.UpdatedAt = req.Timestamp
				}
		}
	}
}

func (perm *booleanRecord) update(val bool, time time.Time) bool {
	if(time.After(perm.UpdatedAt)) {
		perm.Ok = val
		perm.UpdatedAt = time
		return true
	}
	return false
}

func (keyRec *keyRecord) update(val rsa.PublicKey, time time.Time) bool {
	if(time.After(keyRec.UpdatedAt)) {
		keyRec.Key = val
		keyRec.UpdatedAt = time
		return true
	}
	return false
}