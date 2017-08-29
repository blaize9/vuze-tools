package vuze

import (
	"fmt"
	"github.com/blaize9/vuze-tools/config"
	"github.com/blaize9/vuze-tools/utils"
	"github.com/blaize9/vuze-tools/utils/log"
	"github.com/djherbis/times"
	torrentParser "github.com/j-muller/go-torrent-parser"
	"io/ioutil"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type HashStorage struct {
	BackupDirectories []string
	HashMap           HashMap
	LastModified      time.Time
}

type HashMap map[string]FilepathSlice

type FilepathSlice []Filepath

type Filepath struct {
	Filepath     string
	DateModified time.Time
}

type HashMapSorter struct {
	hashes []FilepathSlice
}

func (f FilepathSlice) Sort() {
	sort.Sort(f)
}

func (f FilepathSlice) Len() int {
	return len(f)
}

func (f FilepathSlice) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f FilepathSlice) Less(i, j int) bool {
	return f[i].DateModified.After(f[j].DateModified)
}

func BackupHashFinder(vuzeBackupDirectories *[]string) HashStorage {
	var HashStorage = HashStorage{BackupDirectories: *vuzeBackupDirectories}
	HashStoragePath := filepath.Join(config.GetAzRecoverPath(), "hashstorage.struct")
	var ResumeHashStorage bool
	var NewBackupDirectories []string
	if utils.FileExists(HashStoragePath) {
		fmt.Printf("Checking existing hashstorage.struct\n")
		FileHashStorage, _, hashStorageDirCount, _ := CheckHashStorage()
		fmt.Printf("Current Backup Dirs: %d\nHashStorage File Dirs: %d\n", len(*vuzeBackupDirectories), hashStorageDirCount)

		for _, dir := range *vuzeBackupDirectories {
			if !utils.SliceContains(FileHashStorage.BackupDirectories, dir) {
				NewBackupDirectories = append(NewBackupDirectories, dir)
				fmt.Printf("%s was not found in HashStorage file\n", dir)
			}
		}
		fmt.Println()

		if utils.AskForconfirmation("Would you like to load hashstorage.struct?") {
			if len(*vuzeBackupDirectories) != len(FileHashStorage.BackupDirectories) {
				if utils.AskForconfirmation("Would you like to scan new directories?") {
					ResumeHashStorage = true
					HashStorage = FileHashStorage
				} else {
					return FileHashStorage
				}
			} else {
				return FileHashStorage
			}
		} else {
			HashStorage.BackupDirectories = *vuzeBackupDirectories
		}
	}

	var wg sync.WaitGroup
	var BackupDirectories []string
	if ResumeHashStorage {
		BackupDirectories = NewBackupDirectories
		HashStorage.BackupDirectories = append(HashStorage.BackupDirectories, NewBackupDirectories...)
		HashStorage.BackupDirectories = utils.UniqueStringSlice(HashStorage.BackupDirectories)
	} else {
		BackupDirectories = *vuzeBackupDirectories
	}

	wg.Add(len(BackupDirectories))
	start := time.Now()
	var mutex = &sync.Mutex{}

	var hashMap = make(map[string]FilepathSlice)
	workers := 0
	for _, bkdir := range BackupDirectories {
		for workers > config.Get().AdvancedRecoverMaxWorkers {
			time.Sleep(time.Second * 25)
		}
		workers++
		go func(bkdir string) {
			defer wg.Done()
			torrentDir := filepath.Join(bkdir, config.Get().AzureusTorrentsDirectory)
			if utils.DirExists(torrentDir) {
				files, _ := ioutil.ReadDir(torrentDir + "/")
				for _, tfile := range files {
					if filepath.Ext(tfile.Name()) == ".torrent" {
						tfilepath := filepath.Join(torrentDir, tfile.Name())
						torrent, err := torrentParser.ParseFromFile(tfilepath)
						if err != nil {
							continue
						}
						ftime, _ := times.Stat(tfilepath)
						mutex.Lock()
						hashMap[torrent.InfoHash] = append(hashMap[torrent.InfoHash], Filepath{Filepath: tfilepath, DateModified: ftime.ModTime()})
						mutex.Unlock()
					}
				}
				log.Infof("[W%s] Finished scanning %s (%d files)\n", time.Since(start), torrentDir, len(files))
			}
			workers--
		}(bkdir)
	}
	wg.Wait()
	HashStorage.LastModified = time.Now()
	HashStorage.HashMap = hashMap
	err := utils.SaveStruct(HashStoragePath, HashStorage)
	if err != nil {
		log.Errorf("Error saving hashstorage [%s]", err)
	}

	log.Infof("Total time taken to scan %s", time.Since(start).String())
	return HashStorage
}

func TorrentFinderWorker(worker int, recovered chan<- int, unrecovered chan<- int, torrentFiles <-chan string, finished chan<- bool, chFilesCompleted chan<- int, recoveredMap chan<- RecoveredTorrent, vuzeBackupDirectories *[]string) {
	log.Infof("Worker %d started", worker)

	//defer wg.Done()

	for torrentFilepath := range torrentFiles {
		tfilepath := torrentFilepath
		log.Debugf("Worker %d Working on %s\n", worker, tfilepath)
		filename := filepath.Base(tfilepath)
		var foundTorrentFile bool
		if !utils.FileExists(torrentFilepath) {
			for _, bkdir := range *vuzeBackupDirectories {
				findTorrent := filepath.Join(bkdir, config.Get().AzureusTorrentsDirectory, filename)
				if utils.FileExists(findTorrent) && utils.IsTorrentValid(findTorrent) == nil {
					log.Debugf("[%d] %s FOUND\n", worker, findTorrent)
					recoveredMap <- RecoveredTorrent{Filename: filename, OrigFilepath: torrentFilepath, BackupFilepath: findTorrent}
					recovered <- 1
					foundTorrentFile = true
					break
				}
			}
		}
		if foundTorrentFile == false {
			log.Warnf("[W%d] %s NOT FOUND", worker, tfilepath)
			unrecovered <- 1
		}
		chFilesCompleted <- 1
	}
	log.Infof("Worker %d Closed", worker)
}
