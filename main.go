package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/IncSW/go-bencode"
	"github.com/blaize9/vuze-tools/config"
	"github.com/blaize9/vuze-tools/utils"
	"github.com/blaize9/vuze-tools/utils/log"
	"github.com/blaize9/vuze-tools/vuze"
	"github.com/mitchellh/go-ps"
	pbar "github.com/pmalek/pb"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"
)

var azureusBackupDirectories []string

// TODO: Add Tests
// TODO: Add Documentation

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	config.BindFLags()
	log.Init(config.Get().Environment)

	log.Debugf("Config: %v", config.Get())
	log.Infof("Using %d CPUs", runtime.NumCPU())
	log.Infof("Azureus Directory: %s", config.Get().AzureusDirectory)
	log.Infof("Recovery Directory: %s", config.GetAzRecoverPath())

	if !utils.FileExists(config.GetAzDownloadsConfig()) {
		log.Fatalf("Azureus downloads.config in %s does not exist. Have you set your Azureus Directory?", config.Get().AzureusDirectory)
	}

	if !utils.DirExists(config.GetAzRecoverPath()) {
		os.MkdirAll(config.GetAzRecoverPath(), os.FileMode(0644))
	}

	if !utils.DirExists(filepath.Join(config.GetAzRecoverPath(), "torrents")) {
		os.Mkdir(filepath.Join(config.GetAzRecoverPath(), "torrents"), os.FileMode(0644))
	}

	if !utils.DirExists(filepath.Join(config.GetAzRecoverPath(), "active")) {
		os.Mkdir(filepath.Join(config.GetAzRecoverPath(), "active"), os.FileMode(0644))
	}

	processes, _ := ps.Processes()
	for _, process := range processes {
		if strings.Contains(strings.ToLower(process.Executable()), "azureus") {
			if !utils.AskForconfirmation(fmt.Sprintf("Found (%d) %s running. Would you like to continue?", process.Pid(), process.Executable())) {
				os.Exit(2)
			}
		}
	}

	if len(config.Get().AzureusBackupDirectories) == 0 {
		log.Infof("You have not entered any backup directories to search. Please add them if you want to run Simple or Advanced recoveries.\n")
	}

	for _, directories := range config.Get().AzureusBackupDirectories {
		if directories.Directory == "" {
			continue
		}
		for _, directory := range vuze.GetAllVuzeBackupDirectores(directories.Directory) {
			if !utils.DirExists(directory) {
				continue
			}
			azureusBackupDirectories = append(azureusBackupDirectories, directory)
		}
	}

	vuze.ShuffleBackupDirectories(azureusBackupDirectories)
	azureusBackupDirectories = append([]string{config.Get().AzureusDirectory}, azureusBackupDirectories...)
	azureusBackupDirectories = utils.UniqueStringSlice(azureusBackupDirectories)

	reader := bufio.NewReader(os.Stdin)
	fmt.Println()
	fmt.Printf("Please select a program to run\n" +
		"1. Fix Active files (Scans active for dat files and attempts to fix them)\n" +
		"2. Simple Recovery (Scans backups for missing torrents by filename)\n" +
		"3. Advanced Recovery (Scans backup's torrents and recovers them using hashes.) *Long*\n" +
		"4. Active Recovery (Scans backup.config and recovers torrents from active.dat files) *Fast and accurate*\n" +
		"5. Exit\nSelection: ")
	selection, _ := reader.ReadString('\n')
	selection = strings.TrimSpace(selection)

	switch selection {
	case "1":
		FixActiveDatFiles()
	case "2":
		SimpleRecover()
	case "3":
		AdvancedRecover()
	case "4":
		ActiveRecover()
	default:
		fmt.Println("Exiting")
		os.Exit(2)
	}

	fmt.Println("Press enter key to exit.")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func FixActiveDatFiles() {
	log.Info("Fix Active Dat Files\n-------------------------------")

	dir := config.GetAzActivePath()
	fixedDir := path.Join(config.GetAzRecoverPath(), "active")

	hashes := vuze.ProcessActiveDirectory(config.GetAzActivePath())
	totalHashes := len(hashes)
	bar := pbar.StartNew(totalHashes)

	valid := 0
	recovered := 0
	unrecoverable := 0

	for hash, m := range hashes {
		bar.Increment()
		if m.IsDatValid && m.IsBakValid {
			valid++
		} else {
			// .dat -> .dat.bak
			if m.IsDatValid && !m.IsBakValid {
				recovered++
				err := utils.CopyFile(path.Join(dir, hash+".dat"), path.Join(fixedDir, hash+".dat.bak"))
				if err != nil {
					fmt.Println(err)
				}
				err = utils.CopyFile(path.Join(dir, hash+".dat"), path.Join(fixedDir, hash+".dat"))
				if err != nil {
					fmt.Println(err)
				}
				continue
			}
			// .dat.bak -> .dat
			if m.IsBakValid && !m.IsDatValid {
				recovered++
				err := utils.CopyFile(path.Join(dir, hash+".dat.bak"), path.Join(fixedDir, hash+".dat"))
				if err != nil {
					fmt.Println(err)
				}
				err = utils.CopyFile(path.Join(dir, hash+".dat.bak"), path.Join(fixedDir, hash+".dat.bak"))
				if err != nil {
					fmt.Println(err)
				}
				continue
			}
			// .dat._AZ -> .dat .dat.bak
			if m.IsAZValid && !m.IsBakValid && !m.IsDatValid {
				recovered++
				err := utils.CopyFile(path.Join(dir, hash+".dat._AZ"), path.Join(fixedDir, hash+".dat"))
				if err != nil {
					fmt.Println(err)
				}
				err = utils.CopyFile(path.Join(dir, hash+".dat._AZ"), path.Join(fixedDir, hash+".dat.bak"))
				if err != nil {
					fmt.Println(err)
				}
				continue
			}
			if m.IsSavingValid && !m.IsDatValid && !m.IsBakValid {
				recovered++
				err := utils.CopyFile(path.Join(dir, hash+".dat.saving"), path.Join(fixedDir, hash+".dat"))
				if err != nil {
					fmt.Println(err)
				}
				err = utils.CopyFile(path.Join(dir, hash+".dat.saving"), path.Join(fixedDir, hash+".dat.bak"))
				if err != nil {
					fmt.Println(err)
				}
				continue
			}
			if !m.IsDatValid && !m.IsBakValid && !m.IsAZValid && !m.IsSavingValid {
				unrecoverable++
				log.Warnf("%s is unrecoverable [%v]\n", hash, m)
			} else {
				unrecoverable++
				log.Warnf("%s is unrecoverable [%v]\n", hash, m)
			}
		}
	}
	bar.FinishPrint("Recovery Finished! Please copy the files from " + config.GetAzRecoverPath())

	log.Infof("Total: %d, Valid: %d, Recoverable: %d Unrecoverable: %d", valid, totalHashes, recovered, unrecoverable)
}

func SimpleRecover() {
	log.Info("Simple Recovery\n-------------------------------")
	recoveredMap := make(chan vuze.RecoveredTorrent, 1)
	chFinished := make(chan bool)
	chRecovered := make(chan int)
	chUnrecoverable := make(chan int)
	chTorrentFiles := make(chan string, 4550)
	chFilesCompleted := make(chan int)

	log.Infof("Scanning Downloads config")
	torrents, err := vuze.ScanDownloadsConfig()
	if err != nil {
		log.Fatalf("%v", err)
		return
	}

	torrentFiles := []string{}
	files := 0
	valid := 0
	for _, torrent := range torrents {
		if torrent.Valid && torrent.Found {
			valid++
			continue
		}
		files++
		torrentFiles = append(torrentFiles, torrent.Filepath)
	}

	log.Infof("Sending %d torrents to Torrent Finder Worker", len(torrentFiles))
	go func() {
		for _, torrent_filepath := range torrentFiles {
			for len(chTorrentFiles) == cap(chTorrentFiles) {
				time.Sleep(time.Millisecond * 100)
			}
			chTorrentFiles <- torrent_filepath
		}
	}()

	files_recovered_map := map[string]vuze.RecoveredTorrent{}
	files_complete := 0
	files_recovered := 0
	files_unrecoverable := 0

	for i := 0; i < config.Get().SimpleRecoverWorkers; i++ {
		go vuze.TorrentFinderWorker(i, chRecovered, chUnrecoverable, chTorrentFiles, chFinished, chFilesCompleted, recoveredMap, &azureusBackupDirectories)
	}
	for {
		if files == files_complete && files_recovered+files_unrecoverable == files_complete {
			break
		}

		select {
		case recovered := <-recoveredMap:
			files_recovered_map[recovered.OrigFilepath] = recovered
		case <-chUnrecoverable:
			files_unrecoverable++
		case <-chRecovered:
			files_recovered++
		case <-chFilesCompleted:
			files_complete++
		}

	}

	log.Infof("Found: %d Recoverable: %d Unrecoverable %d", files, files_recovered, files_unrecoverable)

	data, err := vuze.ReadDownloadsConfig()
	if err != nil {
		fmt.Errorf("%v\n", err)
	}
	datam := data.(map[string]interface{})
	recoverTorrents(&datam, files_recovered_map, false)

}

func AdvancedRecover() {
	log.Info("Advanced Recovery\n-------------------------------")
	HashStorage := vuze.BackupHashFinder(&azureusBackupDirectories)

	log.Infof("Sorting HashStorage Hashes by newest")
	for _, hash := range HashStorage.HashMap {
		sort.Sort(hash)
	}

	log.Infof("Scanning Downloads config")
	torrents, err := vuze.ScanDownloadsConfig()
	if err != nil {
		log.Fatalf("%v\n", err)
		return
	}

	log.Infof("Selecting torrents to recover")

	valid := 0
	recovered := 0
	unrecoverable := 0

	files_recovered_map := map[string]vuze.RecoveredTorrent{}
	for i, torrent := range torrents {
		if torrent.Valid && torrent.Found {
			valid++
			continue
		}
		if _, ok := HashStorage.HashMap[hex.EncodeToString(torrent.Hash)]; ok {
			recovered++
			log.Infof("[%d] Recovering %s", i, torrent.Filepath, torrent.Valid, torrent.Found)
			first := HashStorage.HashMap[hex.EncodeToString(torrent.Hash)][0]
			files_recovered_map[torrent.Filepath] = vuze.RecoveredTorrent{Filename: filepath.Base(torrent.Filepath), BackupFilepath: first.Filepath}
		} else {
			unrecoverable++
			log.Warnf("[%d] Unable to recover %s [Found: %v, Valid: %v]", i, torrent.Filepath, torrent.Found, torrent.Valid)
		}

	}

	log.Infof("Total: %d, Valid: %d, Recovered: %d, Unrecoverable: %d\n", len(torrents), valid, recovered, unrecoverable)

	data, err := vuze.ReadDownloadsConfig()
	if err != nil {
		fmt.Errorf("%v", err)
	}
	datam := data.(map[string]interface{})
	recoverTorrents(&datam, files_recovered_map, false)

}

func ActiveRecover() {
	log.Info("Active Recovery\n-------------------------------")
	log.Infof("Scanning Downloads config")
	files_recovered_map := map[string]vuze.RecoveredTorrent{}
	torrents, err := vuze.ScanDownloadsConfig()
	if err != nil {
		log.Fatalf("%v", err)
		return
	}

	valid := len(torrents)
	recovered := 0
	unrecoverable := 0

	for _, torrent := range torrents {
		log.Infof("%v\n", torrent)
		activedat := filepath.Join(config.Get().AzureusDirectory, "active", strings.ToUpper(hex.EncodeToString(torrent.Hash))+".dat")
		log.Infof("Active File: %s\n", activedat)

		if !utils.FileExists(torrent.Filepath) {
			hashstring := strings.ToUpper(hex.EncodeToString(torrent.Hash))
			activedat := filepath.Join(config.Get().AzureusDirectory, "active", hashstring+".dat")
			if utils.FileExists(activedat) {
				saved, err := vuze.SaveTorrentFromActive(activedat, filepath.Join(config.GetAzRecoverPath(), "torrents", filepath.Base(torrent.Filepath)))
				if err != nil {
					log.Errorf("[%s] Unable to Save torrent from active to %s [%v]", hashstring, torrent.Filepath, err)
					unrecoverable++
					continue
				}
				if saved {
					recovered++
					files_recovered_map[torrent.Filepath] = vuze.RecoveredTorrent{Filename: filepath.Base(torrent.Filepath), BackupFilepath: activedat}
				}
			} else {
				log.Warnf("[%s] Unable to find active for %s", hashstring, torrent.Filepath)
				unrecoverable++
			}

		}
	}

	log.Infof("Total: %d, Valid: %d, Recovered: %d, Unrecoverable: %d", len(torrents), valid, recovered, unrecoverable)
}

func recoverTorrents(datam *map[string]interface{}, files_recovered_map map[string]vuze.RecoveredTorrent, updateOnly bool) {
	recoverTorrentsDir := filepath.Join(config.GetAzRecoverPath(), config.Get().AzureusTorrentsDirectory)

	if !updateOnly {
		for _, recovered := range files_recovered_map {
			if recovered.Err != nil {
				continue
			}
			backupfile := recovered.BackupFilepath
			newfile := filepath.Join(recoverTorrentsDir, recovered.Filename)

			if !utils.FileExists(backupfile) {
				log.Errorf("recover file %s does not exist in expected location", backupfile)
			}

			if !utils.FileExists(newfile) && !utils.FileExists(filepath.Join(config.GetAzTorrentsPath(), recovered.Filename)) {
				err := utils.CopyFile(backupfile, newfile)
				if err != nil {
					log.Warnf("Unable to copy %s to %s [%v]", backupfile, newfile, err.Error())
				}
			}
		}

	}

	for _, v := range *datam {
		for _, vv := range v.([]interface{}) {
			for kkk, vvv := range vv.(map[string]interface{}) {
				if kkk == "torrent" {
					torrent_filepath := utils.ByteToString(vvv.([]uint8))
					if recovered, ok := files_recovered_map[torrent_filepath]; ok {
						if recovered.Err != nil {
							continue
						}
						newfile := filepath.Join(config.GetAzTorrentsPath(), recovered.Filename)
						reflect.ValueOf(vv).SetMapIndex(reflect.ValueOf("torrent"), reflect.ValueOf([]uint8(newfile)))
					}
				}
			}
		}
	}

	dataMarshal, err := bencode.Marshal(*datam)
	if err != nil {
		log.Fatal("Unable to marshal downloads.config")
	}

	err = ioutil.WriteFile(filepath.Join(config.GetAzRecoverPath(), "downloads.config"), dataMarshal, 0644)
	if err != nil {
		log.Errorf("Unabel to write new download config [%v]", err)
	}

	fmt.Println("Recovery Complete")
}
