package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/golangWhatsappCustomSoftware/validator"
	"gopkg.in/gomail.v2"
)

type Config struct {
	SendAttachment        bool   `json:"sendAttachment" validate:"boolean"`
	ReadAttachmentFromCSV bool   `json:"readAttachmentFromCSV" validate:"boolean"`
	FixedAttachmentName   string `json:"fixedAttachmentName"`
	BasePathForAttachment string `json:"basePathForAttachment"`
	AddDelayBetweenMails  int64  `json:"addDelayBetweenMails" validate:"min=0,max=10"`
	ReadSubjectFromCsv    bool   `json:"readSubjectFromCsv" validate:"boolean"`
	EmailSubject          string `json:"emailSubject"`
	ReadEmailTextFromCsv  bool   `json:"readEmailTextFromCsv" validate:"boolean"`
	IsEmailBodyHTML       bool   `json:"isEmailBodyHTML" validate:"boolean"`
	EmailBody             string `json:"emailBody" `
	ParallelEmails        int    `json:"parallelEmails" validate:"min=2"`
	InputFileName         string `json:"inputFile"`
	Host                  string `json:"host" validate:"required"`
	Port                  int    `json:"port" validate:"required,port"`
	UserName              string `json:"userName" validate:"required"`
	Password              string `json:"password" validate:"required"`
	FromEmail             string `json:"fromEmail" validate:"required"`
	FromName              string `json:"fromName" validate:"required"`
}

var currentDir = ""
var InputFilePath = ""
var OutPutFilePath = ""
var ThisConfig = new(Config)
var NonNumber, _ = regexp.Compile(`/\D/g`)

func AppendToOutPutFile(text string) {
	f, err := os.OpenFile(OutPutFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err = f.WriteString(text); err != nil {
		panic(err)
	}
}

// func AfterSuccessFullConnection() {
// 	time.Sleep(3 * time.Second)
// 	fmt.Printf("Reading File %s\n", InputFilePath)
// 	inputBytes, err := os.ReadFile(InputFilePath)
// 	check(err)
// 	input := string(inputBytes)
// 	RowsData := strings.Split(input, "\n")
// 	fmt.Printf("total %d Rows Found\n", len(RowsData))
// 	for _, row := range RowsData {
// 		cols := strings.Split(row, ",")
// 		if len(cols) < 2 {
// 			AppendToOutPutFile("Cells Lenght < 2 Found\n")
// 			continue
// 		}
// 		number := string(NonNumber.ReplaceAll([]byte(cols[0]), []byte("")))
// 		if len(number) < 10 {
// 			AppendToOutPutFile(fmt.Sprintf("%s,false,Length %d of Number is Less than 10\n", number, len(number)))
// 			continue
// 		}
// 		fileName := fmt.Sprintf("%s.pdf", strings.TrimSpace(cols[1]))
// 		sendFilePath := filepath.Join(ThisConfig.BasePathForAssets, fileName)
// 		if _, err := os.Stat(sendFilePath); errors.Is(err, os.ErrNotExist) {
// 			AppendToOutPutFile(fmt.Sprintf("%s,false,File Path Not Exists %s\n", number, sendFilePath))
// 			continue
// 		}
// 		IsOnWhatsappCheck, err := client.IsOnWhatsApp([]string{"+" + number})
// 		if err != nil {
// 			AppendToOutPutFile(fmt.Sprintf("%s,false,Soemthinh Went Wrong %#v\n", number, err))
// 			continue
// 		}
// 		NumberOnWhatsapp := IsOnWhatsappCheck[0]
// 		if !NumberOnWhatsapp.IsIn {
// 			AppendToOutPutFile(fmt.Sprintf("%s,false,Number %s Not On Whatsapp\n", number, number))
// 			continue
// 		}
// 		pdfBytes, err := os.ReadFile(sendFilePath)
// 		if err != nil {
// 			AppendToOutPutFile(fmt.Sprintf("%s,false,Error While Reading File %#v\n", number, err))
// 			continue
// 		}
// 		println("Uploading File")
// 		resp, err := client.Upload(context.Background(), pdfBytes, whatsmeow.MediaDocument)
// 		if err != nil {
// 			AppendToOutPutFile(fmt.Sprintf("%s,false,Error While Uploading %#v\n", number, err))
// 			continue
// 		}
// 		docProto := &waProto.DocumentMessage{
// 			Url:           &resp.URL,
// 			Mimetype:      proto.String("application/pdf"),
// 			FileName:      &fileName,
// 			DirectPath:    &resp.DirectPath,
// 			MediaKey:      resp.MediaKey,
// 			FileEncSha256: resp.FileEncSHA256,
// 			FileSha256:    resp.FileSHA256,
// 			FileLength:    &resp.FileLength,
// 		}

// 		if ThisConfig.AppendMessageToMedia {
// 			if !ThisConfig.ReadMessageFromCsv {
// 				docProto.Caption = &ThisConfig.Message
// 			} else if ThisConfig.ReadMessageFromCsv && len(cols) >= 3 && len(cols[2]) > 0 {
// 				docProto.Caption = &cols[2]
// 			}
// 		}
// 		// targetJID := types.NewJID("917016879936", types.DefaultUserServer)
// 		targetJID := NumberOnWhatsapp.JID
// 		fmt.Printf("sending File To %s\n", number)
// 		client.SendMessage(context.TODO(), targetJID, &waProto.Message{
// 			DocumentMessage: docProto,
// 		})
// 		if !ThisConfig.AppendMessageToMedia {
// 			message := new(string)
// 			if !ThisConfig.ReadMessageFromCsv {
// 				message = &ThisConfig.Message
// 			} else if ThisConfig.ReadMessageFromCsv && len(cols) >= 3 && len(cols[2]) > 0 {
// 				message = &cols[2]
// 			}
// 			if len(*message) > 0 {
// 				fmt.Printf("sending Message To %s\n", number)
// 				client.SendMessage(context.TODO(), targetJID, &waProto.Message{
// 					Conversation: proto.String(*message),
// 				})
// 			}
// 		}
// 		AppendToOutPutFile(fmt.Sprintf("%s,true\n", number))
// 		time.Sleep(time.Second * time.Duration(ThisConfig.AddMinimumDelayInSecondsAfterSuccessfullMesssage))
// 	}
// }

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	// using the function
	fmt.Println(len(os.Args), os.Args)
	if slices.Contains(os.Args, "--dev") {
		curren, err := os.Getwd()
		check(err)
		currentDir = curren
	} else {
		exePath, err := os.Executable()
		currentDir = filepath.Dir(exePath)
		check(err)
	}
	configFilePAth := filepath.Join(currentDir, "configs.json")
	fmt.Printf("Config path %s\n", configFilePAth)
	if _, err := os.Stat(configFilePAth); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("Config Not Exist on Path %s", configFilePAth))
	}
	dat, err := os.ReadFile(configFilePAth)
	check(err)
	json.Unmarshal(dat, ThisConfig)

	if errs := validator.Validator.Validate(ThisConfig); len(errs) > 0 {
		panic(fmt.Errorf("Config Error %#v", errs))
	}
	if ThisConfig.SendAttachment && !ThisConfig.ReadAttachmentFromCSV && len(ThisConfig.FixedAttachmentName) < 2 {
		panic("Please Pass Email Attachment in Config Or Make ReadAttachmentFromCSV true")
	}
	if !ThisConfig.ReadSubjectFromCsv && len(ThisConfig.EmailSubject) < 2 {
		panic("Please Pass Email Subject in Config Or Make ReadSubjectFromCsv true")
	}
	if !ThisConfig.ReadEmailTextFromCsv && len(ThisConfig.EmailBody) < 2 {
		panic("Please Pass Email Body in Config Or Make ReadEmailTextFromCsv true")
	}
	if ThisConfig.BasePathForAttachment == "" {
		ThisConfig.BasePathForAttachment = filepath.Join(currentDir, "assets")
	}

	if _, err := os.Stat(ThisConfig.BasePathForAttachment); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("base path for assets not exists %s", configFilePAth))
	}
	if len(ThisConfig.InputFileName) > 0 {
		InputFilePath = filepath.Join(currentDir, ThisConfig.InputFileName)
	} else {
		InputFilePath = filepath.Join(currentDir, "input.csv")
	}
	OutPutFilePath = filepath.Join(filepath.Dir(InputFilePath), "output.csv")
	if _, err := os.Stat(InputFilePath); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("input File Not Exists at %s", InputFilePath))
	}
	Emails()
}

func Emails() {
	d := gomail.NewDialer(ThisConfig.Host, ThisConfig.Port, ThisConfig.UserName, ThisConfig.Password)
	authenticated, err := d.Dial()
	defer func() { authenticated.Close() }()
	if err != nil {
		panic(err)
	}
	time.Sleep(3 * time.Second)
	fmt.Printf("Reading File %s\n", InputFilePath)
	inputBytes, err := os.ReadFile(InputFilePath)
	check(err)
	input := string(inputBytes)
	_, err = os.Create(OutPutFilePath) //create a new file
	if err != nil {
		fmt.Println(err)
		return
	}
	RowsData := strings.Split(input, "\n")
	fmt.Printf("total %d Rows Found\n", len(RowsData))
	currentQueCount := 0
	// var wg sync.WaitGroup
	for _, row := range RowsData {
		cols := strings.Split(row, ",")
		checkCells := 1
		if ThisConfig.ReadAttachmentFromCSV {
			checkCells = 2
		}
		if ThisConfig.ReadSubjectFromCsv {
			checkCells = 3
		}
		if ThisConfig.ReadEmailTextFromCsv {
			checkCells = 5
		}
		if len(cols) < checkCells {
			AppendToOutPutFile(fmt.Sprintf("Cells Length < %d Found\n", checkCells))
			continue
		}
		address := strings.TrimSpace(cols[0])
		if _, err := mail.ParseAddress(address); err != nil {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Invalid Email\n", address))
			continue
		}
		attacheMentName := strings.TrimSpace(ThisConfig.FixedAttachmentName)
		if ThisConfig.ReadAttachmentFromCSV {
			attacheMentName = strings.TrimSpace(cols[1])
		}
		email := gomail.NewMessage()
		if attacheMentName != "" {
			attacheMentNamePath := filepath.Join(ThisConfig.BasePathForAttachment, attacheMentName)
			if _, err := os.Stat(attacheMentNamePath); errors.Is(err, os.ErrNotExist) {
				AppendToOutPutFile(fmt.Sprintf("%s,false,File Path Not Exists %s\n", address, attacheMentNamePath))
				continue
			}
			email.Attach(attacheMentNamePath)
		}
		emailSubject := ThisConfig.EmailSubject
		if ThisConfig.ReadSubjectFromCsv {
			emailSubject = strings.TrimSpace(cols[2])
		}
		if emailSubject == "" {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Email Subject Not Found\n", address))
			continue
		}
		email.SetHeaders(map[string][]string{
			"From":    {email.FormatAddress(ThisConfig.FromEmail, ThisConfig.FromName)},
			"To":      {address},
			"Subject": {emailSubject},
		})
		emailBody := ThisConfig.EmailBody
		if ThisConfig.ReadEmailTextFromCsv {
			emailBody = strings.TrimSpace(cols[3])
		}
		if emailBody == "" {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Email Body Not Found\n", address))
			continue
		}
		isHtml := ThisConfig.IsEmailBodyHTML
		if ThisConfig.ReadEmailTextFromCsv {
			localBody := strings.ToLower(strings.TrimSpace(cols[4]))
			if localBody == "true" {
				isHtml = true
			}
		}
		if isHtml {
			email.SetBody("text/html", emailBody)
		} else {
			email.SetBody("text/plain", emailBody)
		}
		currentQueCount++
		// wg.Add(1)
		// go func() {
		fmt.Printf("Sending Mail to %s\n", address)

		// Send Mail

		err := gomail.Send(authenticated, email)
		if err != nil {
			fmt.Printf("Mail Sending to %s failed \n", address)
			AppendToOutPutFile(fmt.Sprintf("%s,false,Something Went Wrong %#v\n", address, err))
		} else {
			fmt.Printf("Mail Send to %s\n", address)
			AppendToOutPutFile(fmt.Sprintf("%s,true\n", address))
		}

		// wg.Done()
		// }()
		if currentQueCount == ThisConfig.ParallelEmails {
			currentQueCount = 0
			// wg.Wait()
		}
	}
	// wg.Wait()
	// close()
}
