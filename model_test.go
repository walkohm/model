package model_test

import (
	"context"
	"sort"
	"testing"

	"github.com/go-qbit/model"
	"github.com/go-qbit/model/expr"
	"github.com/go-qbit/model/test"

	"github.com/stretchr/testify/suite"
)

type ModelTestSuite struct {
	suite.Suite
	storage *test.Storage
	user    *test.User
	phone   *test.Phone
	address *test.Address
	message *test.Message
}

func TestModelTestSuite(t *testing.T) {
	suite.Run(t, new(ModelTestSuite))
}

func (s *ModelTestSuite) SetupTest() {
	s.storage = test.NewStorage()

	s.user = test.NewUser(s.storage)
	s.phone = test.NewPhone(s.storage)
	s.message = test.NewMessage(s.storage)
	s.address = test.NewAddress(s.storage)

	model.AddOneToOneRelation(s.phone, s.user, false)
	model.AddManyToOneRelation(s.message, s.user)
	model.AddManyToManyRelation(s.user, s.address, s.storage)

	_, err := s.user.AddFromStructs(context.Background(),
		[]struct {
			Id       int
			Name     string
			Lastname string
		}{
			{1, "Ivan", "Sidorov"},
			{2, "Petr", "Ivanov"},
			{3, "James", "Bond"},
			{4, "John", "Connor"},
			{5, "Sara", "Connor"},
		},
	)

	s.NoError(err)

	_, err = s.phone.AddMulti(context.Background(),
		[]string{"id", "country_code", "code", "number"},
		[][]interface{}{
			{1, 1, 111, 1111111},
			{3, 3, 333, 3333333},
		},
	)
	s.NoError(err)

	_, err = s.message.AddMulti(context.Background(),
		[]string{"id", "text", "fk__user__id"},
		[][]interface{}{
			{10, "Message 1", 1},
			{20, "Message 2", 1},
			{30, "Message 3", 1},
			{40, "Message 4", 2},
		},
	)
	s.NoError(err)

	_, err = s.address.AddMulti(context.Background(),
		[]string{"id", "country", "city", "address"},
		[][]interface{}{
			{100, "USA", "Arlington", "1022 Bridges Dr"},
			{200, "USA", "Fort Worth", "7105 Plover Circle"},
			{300, "USA", "Crowley", "524 Pecan Street"},
			{400, "USA", "Arlington", "1022 Bridges Dr"},
			{500, "USA", "Louisville", "1246 Everett Avenue"},
		},
	)
	s.NoError(err)

	_, err = s.user.GetRelation("address").JunctionModel.AddMulti(context.Background(),
		[]string{"fk__user__id", "fk__address__id"},
		[][]interface{}{
			{1, 100},
			{1, 200},
			{2, 200},
			{2, 300},
			{3, 300},
			{4, 400},
			{5, 500},
		})
	s.NoError(err)
}

func (s *ModelTestSuite) TestModel_CheckRegisteredModels() {
	s.Equal(
		[]string{"_junction__user__address", "address", "message", "phone", "user"},
		s.storage.GetModelsNames(),
	)
}

func (s *ModelTestSuite) TestModel_GetFieldsNames() {
	s.Equal(
		[]string{"id", "name", "lastname", "fullname"},
		s.user.GetFieldsNames(),
	)

	s.Equal(
		[]string{"id", "text", "fk__user__id"},
		s.message.GetFieldsNames(),
	)
}

func (s *ModelTestSuite) TestModel_GetRelations() {
	relations := s.user.GetRelations()
	sort.Strings(relations)

	s.Equal(
		[]string{"address", "message", "phone"},
		relations,
	)
}

func (s *ModelTestSuite) TestModel_GetAll() {
	data, err := s.user.GetAll(
		context.Background(),
		[]string{
			// Self fields
			"id", "lastname", "fullname",
			// 1 to 1
			"phone.formated_number",
			// 1 to many
			"message.text",
			// many to many
			"address.city", "address.address",
		},
		model.GetAllOptions{
			Filter: expr.Lt(expr.ModelField("id"), expr.Value(4)),
		},
	)
	s.NoError(err)

	s.Equal([]map[string]interface{}{
		{
			"id":       1,
			"lastname": "Sidorov",
			"fullname": "Ivan Sidorov",
			"phone":    map[string]interface{}{"formated_number": "+1 (111) 1111111"},
			"message": []map[string]interface{}{
				{"text": "Message 1"},
				{"text": "Message 2"},
				{"text": "Message 3"},
			},
			"address": []map[string]interface{}{
				{"city": "Arlington", "address": "1022 Bridges Dr"},
				{"city": "Fort Worth", "address": "7105 Plover Circle"},
			},
		},
		{
			"id":       2,
			"lastname": "Ivanov",
			"fullname": "Petr Ivanov",
			"message": []map[string]interface{}{
				{"text": "Message 4"},
			},
			"address": []map[string]interface{}{
				{"city": "Fort Worth", "address": "7105 Plover Circle"},
				{"city": "Crowley", "address": "524 Pecan Street"},
			},
		},
		{
			"id":       3,
			"lastname": "Bond",
			"fullname": "James Bond",
			"phone":    map[string]interface{}{"formated_number": "+3 (333) 3333333"},
			"address": []map[string]interface{}{
				{"city": "Crowley", "address": "524 Pecan Street"},
			},
		},
	}, data)
}

func (s *ModelTestSuite) TestModel_GetAllToStruct() {
	type PhoneType struct {
		FormatedNumber string
	}

	type MessageType struct {
		Text string
	}

	type AddressTYpe struct {
		City    string
		Address string
	}

	type UserType struct {
		Id        int           `field:"id"`
		Lastname  string        `field:"lastname"`
		Fullname  string        `field:"fullname"`
		Phone     PhoneType     `field:"phone"`
		PhonePtr  *PhoneType    `field:"phone"`
		Messages  []MessageType `field:"message"`
		Addresses []AddressTYpe `field:"address"`
	}

	var res []UserType

	s.NoError(s.user.GetAllToStruct(
		context.Background(),
		&res,
		model.GetAllOptions{
			Filter: expr.Lt(expr.ModelField("id"), expr.Value(4)),
		},
	))

	s.Equal(
		[]UserType{
			{
				Id:       1,
				Lastname: "Sidorov",
				Fullname: "Ivan Sidorov",
				Phone: PhoneType{
					FormatedNumber: "+1 (111) 1111111",
				},
				PhonePtr: &PhoneType{
					FormatedNumber: "+1 (111) 1111111",
				},
				Messages: []MessageType{
					{Text: "Message 1"},
					{Text: "Message 2"},
					{Text: "Message 3"},
				},
				Addresses: []AddressTYpe{
					{City: "Arlington", Address: "1022 Bridges Dr"},
					{City: "Fort Worth", Address: "7105 Plover Circle"},
				},
			},
			{
				Id:       2,
				Lastname: "Ivanov",
				Fullname: "Petr Ivanov",
				Messages: []MessageType{
					{Text: "Message 4"},
				},
				Addresses: []AddressTYpe{
					{City: "Fort Worth", Address: "7105 Plover Circle"},
					{City: "Crowley", Address: "524 Pecan Street"},
				},
			},
			{
				Id:       3,
				Lastname: "Bond",
				Fullname: "James Bond",
				Phone: PhoneType{
					FormatedNumber: "+3 (333) 3333333",
				},
				PhonePtr: &PhoneType{
					FormatedNumber: "+3 (333) 3333333",
				},
				Addresses: []AddressTYpe{
					{City: "Crowley", Address: "524 Pecan Street"},
				},
			},
		},
		res,
	)
}
