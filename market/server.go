package market

import (
	"log"
	"sync"
	"time"
)

type Server struct {
	lock       sync.Mutex
	status     int
	Logger     *log.Logger
	LogPath    string
	ServerName string
}

func (s *Server) GetStatus() (st int) {
	s.lock.Lock()
	st = s.status
	s.lock.Unlock()
	return
}

func (s *Server) start(ch chan int, wg *sync.WaitGroup) {
	_logger := s.Logger
	if _logger == nil {
		panic("no logger in this server")
	}
	var (
		v   MDataMap
		err error
		st  time.Time
	)
	//fmt.Println("2", s.ServerName, s.LogPath, s.Logger, s)
	_logger.Println("Thread started at", time.Now().Format(time.RFC3339))
	s.lock.Lock()
	s.status = 0
	s.lock.Unlock()
	ch <- 0
	for kk := 0; kk < 10; kk++ {
		_logger.Println("Start processing market data at", time.Now().Format(time.RFC3339))
		s.lock.Lock()
		s.status = 1
		s.lock.Unlock()
		ch <- 1
		st = time.Now()
		v, err = MktRequestsDistributor(s.ServerName, 10000002)
		if err != nil {
			s.lock.Lock()
			s.status = -1
			s.lock.Unlock()
			ch <- -1
			_logger.Panicln("Error occurred: ", err)
		} else {
			err = v.DatabaseUpdate(s.ServerName)
			if err != nil {
				s.lock.Lock()
				s.status = -1
				s.lock.Unlock()
				ch <- -1
				_logger.Panicln("Error occurred: ", err)
			} else {
				_logger.Println("Successfully updated Market Database")
				s.lock.Lock()
				s.status = 2
				s.lock.Unlock()
				ch <- 2
			}
		}
		_logger.Printf("Finish Database Updating. Using time: %s\n", time.Now().Sub(st).String())
		s.lock.Lock()
		s.status = 0
		s.lock.Unlock()
		ch <- 0
		time.Sleep(time.Minute * 10)
	}
	wg.Done()
}

func (s *Server) Start(send bool, ech chan int, wg *sync.WaitGroup) {
	var (
		ec chan int
	)
	if !send {
		ec = make(chan int, 1)
	} else {
		ec = ech
	}
	go s.start(ec, wg)
}
