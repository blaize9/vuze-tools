package utils

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/IncSW/go-bencode"
	"github.com/djherbis/times"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

func FileExists(Path string) bool {
	if _, err := os.Stat(Path); os.IsNotExist(err) {
		return false
	}
	return true
}

func DirExists(Path string) bool {
	return FileExists(Path)
}

func GetAllSubDirectories(Path string) (dirs []string) {
	files, _ := ioutil.ReadDir(Path)
	for _, f := range files {
		if f.IsDir() {
			dirs = append(dirs, filepath.Join(Path, f.Name()))
		}
	}
	return dirs
}

func CopyFile(Filepath string, destFilepath string) error {
	srcFile, err := os.Open(Filepath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcStat, err := times.Stat(Filepath)
	if err != nil {
		return err
	}

	destFile, err := os.Create(destFilepath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	err = destFile.Sync()

	if err := os.Chtimes(destFilepath, srcStat.AccessTime(), srcStat.ModTime()); err != nil {
		return err
	}

	return nil
}

func CopyFileDateCreated(Filepath string, destFilepath string) error {
	srcFile, err := os.Open(Filepath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcStat, err := times.Stat(Filepath)
	if err != nil {
		return err
	}

	destFile, err := os.Create(destFilepath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	err = destFile.Sync()
	var createdTime time.Time
	if srcStat.HasBirthTime() {
		createdTime = srcStat.BirthTime()
	} else {
		createdTime = srcStat.ModTime()
	}

	if err := os.Chtimes(destFilepath, createdTime, createdTime); err != nil {
		return err
	}

	return nil
}

func IsTorrentValid(filepath string) error {
	file, er := ioutil.ReadFile(filepath)
	if er != nil {
		return errors.New("Read")
	}
	_, err := bencode.Unmarshal(file)
	if err != nil {
		return errors.New("Torrent")
	}
	return nil
}

// Encode via Gob to file
func SaveStruct(path string, object interface{}) error {
	file, err := os.Create(path)
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(object)
	}
	file.Close()
	return err
}

// Decode Gob file
func LoadStruct(path string, object interface{}) error {
	file, err := os.Open(path)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(object)
	}
	file.Close()
	return err
}

func ByteToString(bs []uint8) string {
	b := make([]byte, len(bs))
	for i, v := range bs {
		b[i] = byte(v)
	}
	return string(b)
}

func IsMap(in interface{}) bool {
	va := reflect.ValueOf(in)
	if va.Kind() == reflect.Map {
		return true
	}
	return false
}

func SliceContains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

func AskForconfirmation(msg string) bool {
	var s string

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (Y/N): ", msg)

	s, _ = reader.ReadString('\n')

	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	if s == "y" || s == "yes" {
		return true
	}
	return false
}

func ShuffleSlice(slice []string) {
	for i := range slice {
		j := rand.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
}

func IsBencodeFileValid(path string) bool {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return false
	}
	_, err = bencode.Unmarshal(file)
	if err == nil {
		return true
	}
	return false
}

func UniqueStringSlice(input []string) []string {
	unique := make([]string, 0, len(input))
	existsStringMap := make(map[string]bool)

	for _, string := range input {
		if _, ok := existsStringMap[string]; !ok {
			existsStringMap[string] = true
			unique = append(unique, string)
		}
	}

	return unique
}

func DisplayInterfaceValues(in interface{}) {
	va := reflect.ValueOf(in)
	if va.Kind() == reflect.Map {
		fmt.Printf("%v\n", va)
		m := in.(map[string]interface{})
		for k, v := range m {
			switch vv := v.(type) {
			case []interface{}:
				fmt.Printf("Type: %T\n", vv)
				fmt.Println(k, "is an array:")
				for i, u := range vv {
					fmt.Printf("Type: %T\n", u)
					switch vvv := u.(type) {
					case map[string]interface{}:
						fmt.Println(k, "is map of strings", vvv)
					}
					fmt.Println(i, u)
				}
			default:
				fmt.Printf("Type: %T\n", vv)
			}

		}
	} else {
		m := in.(map[string]interface{})
		for k, v := range m {
			switch vv := v.(type) {
			case string:
				fmt.Println(k, "is string", vv)
			case int:
				fmt.Println(k, "is int", vv)
			case []uint8:
				fmt.Print(k, " is ", ByteToString(vv), "\n")
			case []interface{}:
				fmt.Println(k, "is an array:")
				for i, u := range vv {
					fmt.Println(i)
					DisplayInterfaceValues(u)
				}
			case map[string]interface{}:
				fmt.Println(k, "is map of strings", vv)

			default:
				fmt.Println(k, "is of a type I don't know how to handle ", vv)
			}
		}
	}
}
