package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/golangWhatsappCustomSoftware/validator"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"

	// "go.mau.fi/whatsmeow/types"
	"github.com/denisbrodbeck/machineid"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type AssetsPathArray struct {
	Path  string
	Image bool
	Mime  string
}

type Config struct {
	// LicKey                                         string `json:"licKey" validate:"required"`
	UseTextMessage                                 bool   `json:"useTextMessage" validate:"boolean"`
	AppendMessageToMedia                           bool   `json:"appendMessageToMedia" validate:"boolean"`
	ReadMessageFromCsv                             bool   `json:"readMessageFromCsv" validate:"boolean"`
	Message                                        string `json:"message"`
	AddMinimumDelayInSecondsAfterSuccessfulMessage int    `json:"addMinimumDelayInSecondsAfterSuccessfulMessage" validate:"required"`
	BasePathForAssets                              string `json:"basePathForAssets"`
	InputFileName                                  string `json:"inputFileName"`
}

var currentDir = ""
var InputFilePath = ""
var OutPutFilePath = ""
var ThisConfig = new(Config)
var client *whatsmeow.Client
var NonNumber, _ = regexp.Compile(`/\D/g`)

const AesKey = "SYwHUQteFrNYAf4y4ZimRMm1yZshUsmQ0z0dQnOWXh+1b/pFzhh2ekdhS56SqYF3"

var LoopStarted = false

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message!", v.Message.GetConversation())
	case *events.Connected:
		println("Client Connected")
		go AfterSuccessFullConnection()
	default:
		fmt.Printf("Event Occurred%s\n", reflect.TypeOf(v))
	}
}

func AppendToOutPutFile(text string) {
	println(text)
	f, err := os.OpenFile(OutPutFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(text); err != nil {
		panic(err)
	}
}

func AfterSuccessFullConnection() {
	if LoopStarted {
		println("Tried to Start Loop Again")
		return
	}
	LoopStarted = true
	time.Sleep(3 * time.Second)
	fmt.Printf("Reading File %s\n", InputFilePath)
	inputBytes, err := os.ReadFile(InputFilePath)
	check(err)
	input := string(inputBytes)
	RowsData := strings.Split(input, "\n")
	fmt.Printf("total %d Rows Found\n", len(RowsData))
	for _, row := range RowsData {
		// func() {
		cols := strings.Split(row, ",")
		if len(cols) < 2 {
			AppendToOutPutFile("Cells Length < 2 Found\n")
			continue
		}
		number := string(NonNumber.ReplaceAll([]byte(cols[0]), []byte("")))
		if len(number) < 10 {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Length %d of Number is Less than 10\n", number, len(number)))
			continue
		}
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("panic occured: ", r)
				AppendToOutPutFile(fmt.Sprintf("%s,false,Something Went Wrong %#v\n", number, r))

			}
		}()
		baseFileName := strings.TrimSpace(cols[1])
		assetsPathArray := []AssetsPathArray{
			{
				Path:  filepath.Join(ThisConfig.BasePathForAssets, fmt.Sprintf("%s.pdf", baseFileName)),
				Image: false,
				Mime:  "application/pdf",
			},
			{
				Path:  filepath.Join(ThisConfig.BasePathForAssets, fmt.Sprintf("%s.jpg", baseFileName)),
				Image: true,
				Mime:  "image/jpg",
			},
			{
				Path:  filepath.Join(ThisConfig.BasePathForAssets, fmt.Sprintf("%s.jpeg", baseFileName)),
				Image: true,
				Mime:  "image/jpeg",
			},
			{
				Path:  filepath.Join(ThisConfig.BasePathForAssets, fmt.Sprintf("%s.png", baseFileName)),
				Image: true,
				Mime:  "image/png",
			},
			{
				Path:  filepath.Join(ThisConfig.BasePathForAssets, fmt.Sprintf("%s.webpp", baseFileName)),
				Image: true,
				Mime:  "image/webpp",
			},
		}
		assetsPath := new(AssetsPathArray)
		for _, f := range assetsPathArray {
			if _, err := os.Stat(f.Path); err == nil {
				// path/to/whatever exists
				assetsPath = &f
			} else if errors.Is(err, os.ErrNotExist) {
				continue
			} else {
				continue
			}
		}
		if assetsPath.Path == "" {
			AppendToOutPutFile(fmt.Sprintf("%s,false,File Path Not Exists %s\n", number, baseFileName))
			continue
		}
		IsOnWhatsappCheck, err := client.IsOnWhatsApp([]string{"+" + number})
		if err != nil {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Something Went Wrong %#v\n", number, err))
			continue

		}
		NumberOnWhatsapp := IsOnWhatsappCheck[0]
		if !NumberOnWhatsapp.IsIn {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Number %s Not On Whatsapp\n", number, number))
			continue
		}
		pdfBytes, err := os.ReadFile(assetsPath.Path)
		if err != nil {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Error While Reading File %#v\n", number, err))
			continue
		}
		println("Uploading File")
		mediaType := whatsmeow.MediaDocument
		if assetsPath.Image {
			mediaType = whatsmeow.MediaImage
		}
		resp, err := client.Upload(context.Background(), pdfBytes, mediaType)
		if err != nil {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Error While Uploading %#v\n", number, err))
			continue
		}
		appendMessage := ""
		if ThisConfig.AppendMessageToMedia {
			if !ThisConfig.ReadMessageFromCsv {
				appendMessage = ThisConfig.Message
			} else if ThisConfig.ReadMessageFromCsv && len(cols) >= 3 && len(cols[2]) > 0 {
				appendMessage = cols[2]
			}
		}
		var protoMsg *waE2E.Message
		if assetsPath.Image {
			imageMessage := &waE2E.ImageMessage{
				URL:           &resp.URL,
				Mimetype:      proto.String(assetsPath.Mime),
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileSHA256:    resp.FileSHA256,
				FileEncSHA256: resp.FileEncSHA256,
				FileLength:    &resp.FileLength,
				// JpegThumbnail: pdfBytes,
			}
			if appendMessage != "" {
				imageMessage.Caption = &appendMessage
			}
			protoMsg = &waE2E.Message{
				ImageMessage: imageMessage,
			}
		} else {
			fileName := filepath.Base(assetsPath.Path)
			documnetMessage := &waE2E.DocumentMessage{
				URL:           &resp.URL,
				Mimetype:      proto.String(assetsPath.Mime),
				FileName:      &fileName,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			}
			if appendMessage != "" {
				documnetMessage.Caption = &appendMessage
			}
			protoMsg = &waE2E.Message{
				DocumentMessage: documnetMessage,
			}
		}

		// targetJID := types.NewJID("917016879936", types.DefaultUserServer)
		targetJID := NumberOnWhatsapp.JID
		fmt.Printf("sending File To %s\n", number)
		if !ThisConfig.AppendMessageToMedia {
			message := new(string)
			if !ThisConfig.ReadMessageFromCsv {
				message = &ThisConfig.Message
			} else if ThisConfig.ReadMessageFromCsv && len(cols) >= 3 && len(cols[2]) > 0 {
				message = &cols[2]
			}
			if len(*message) > 0 {
				fmt.Printf("sending Message To %s\n", number)
				client.SendMessage(context.TODO(), targetJID, &waE2E.Message{
					Conversation: proto.String(*message),
				})
			}
		}
		if resp, err := client.SendMessage(context.TODO(), targetJID, protoMsg); err != nil {
			println(err.Error())
		} else {
			println(resp.DebugTimings.Queue)
		}
		AppendToOutPutFile(fmt.Sprintf("%s,true\n", number))
		time.Sleep(time.Second * time.Duration(ThisConfig.AddMinimumDelayInSecondsAfterSuccessfulMessage))
	}
	println("It is Completed")
	os.Exit(0)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	// using the function
	fmt.Println(len(os.Args), os.Args)

	if slices.Contains(os.Args, "--dev") {
		current, err := os.Getwd()
		check(err)
		currentDir = current
	} else {
		exePath, err := os.Executable()
		currentDir = filepath.Dir(exePath)
		check(err)
	}
	configFilePAth := filepath.Join(currentDir, "configs.json")

	if _, err := os.Stat(configFilePAth); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("Config Not Exist on Path %s", configFilePAth))
	}
	dat, err := os.ReadFile(configFilePAth)
	check(err)
	json.Unmarshal(dat, ThisConfig)
	machineId, err := machineid.ID()
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	if errs := validator.Validator.Validate(ThisConfig); len(errs) > 0 {
		fmt.Printf("Machine Id:=%s\n", machineId)
		panic(fmt.Errorf("Config Error %#v", errs))
	}
	if ThisConfig.UseTextMessage {
		if !ThisConfig.ReadMessageFromCsv && len(ThisConfig.Message) == 0 {
			panic("Please Pass Message in Config File If you want to send Text Message Or Make useTextMessage to false")
		}
	}
	// decrypted := decrypt(ThisConfig.LicKey, AesKey)
	// if decrypted != machineId {
	// 	fmt.Printf("Machine Id:=%s\n", machineId)
	// 	panic(fmt.Errorf("Invalid License"))
	// }
	if ThisConfig.BasePathForAssets == "" {
		ThisConfig.BasePathForAssets = filepath.Join(currentDir, "assets")
	}

	if _, err := os.Stat(ThisConfig.BasePathForAssets); errors.Is(err, os.ErrNotExist) {
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
	Whatsapp()
}

func Whatsapp() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	// Make sure you add appropriate DB connector imports, e.g. github.com/mattn/go-sqlite3 for SQLite
	container, err := sqlstore.New("sqlite3", "file:WhatsappSuperSecrete.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "INFO", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println("QR code:", evt.Code)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		println("Connected")
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
func Decrypt(encryptedString string, keyString string) (decryptedString string) {

	key, _ := hex.DecodeString(keyString)
	enc, _ := hex.DecodeString(encryptedString)

	//Create a new Cipher Block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}

	//Create a new GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}

	//Get the nonce size
	nonceSize := aesGCM.NonceSize()

	//Extract the nonce from the encrypted data
	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]

	//Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err.Error())
	}

	return fmt.Sprintf("%s", plaintext)
}
