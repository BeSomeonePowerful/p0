// Implementation of a MultiEchoServer. Students should write their code in this file.

package p0

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"github.com/google/uuid"
)

type multiEchoServer struct {
	lock              chan bool
	connectionManager map[string]*Connection
	done              chan bool
	// TODO: implement this!
}

type Connection struct {
	Id         string
	connection net.Conn
	msgQueue   chan string
	//indicating that connection has been close
	closed chan bool
}

// New creates and returns (but does not start) a new MultiEchoServer.
func New() MultiEchoServer {
	// TODO: implement this!
	lock := make(chan bool, 1)
	lock <- true
	return &multiEchoServer{lock: lock, connectionManager: make(map[string]*Connection), done: make(chan bool)}
}

func sender(con net.Conn, msgQueue chan string, done chan bool) {
	for {
		//is this the best way to write this? Is there any better way?
		finished := false
		select {
		case <-done:
			finished = true
		default:
		}
		if finished {
			break
		}
		msg := <-msgQueue
		con.Write([]byte(msg))
		//go func(msg string) {
		//	con.Write([]byte(msg))
		//}(msg)
	}
}

func (mes *multiEchoServer) AddConn(id string, con net.Conn, done chan bool) {
	<-mes.lock
	queue := make(chan string, 100)
	connection := &Connection{
		Id:         id,
		connection: con,
		msgQueue:   queue,
		closed:     make(chan bool),
	}
	mes.connectionManager[id] = connection
	mes.lock <- true
	go sender(con, queue, done)
}

func (mes *multiEchoServer) Start(port int) error {
	// TODO: implement this!
	// listen on all interfaces
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))

	if err != nil {
		log.Fatal("tcp server listener error:", err)
	}
	go func() {
		// block main and listen to all incoming connections
		for {
			// accept new connection
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("tcp server accept error: %v\n", err)
			}

			// spawn off goroutine to able to accept new connections
			go mes.handleConnection(conn)
		}
	}()
	return nil
}

func (mes *multiEchoServer) Close() {
	close(mes.done)
	<-mes.lock
	for _, item := range mes.connectionManager {
		item.connection.Close()
		close(item.closed)
	}
	mes.lock <- true
	// TODO: implement this!
}

func (mes *multiEchoServer) Count() int {
	// TODO: implement this!
	ret := 0
	<-mes.lock
	ret = len(mes.connectionManager)
	mes.lock <- true
	return ret
}

func (mes *multiEchoServer) RemoveConn(id string) {
	<-mes.lock
	delete(mes.connectionManager, id)
	mes.lock <- true
}

// TODO: add additional methods/functions below!
//whenever a message is send, its first send to channel, from channel, its send to related connections
func (mes *multiEchoServer) handleConnection(conn net.Conn) {
	id := uuid.New().String()
	mes.AddConn(id, conn, mes.done)
	reader := bufio.NewReader(conn)
	defer func() {
		mes.RemoveConn(id)
		conn.Close()
	}()
	for {
		//if mes.hasClosed(id) {
		//	return
		//}
		bufferBytes, err := reader.ReadBytes('\n')

		if err != nil || len(bufferBytes) == 0 {
			log.Println("client left..", err)
			//conn.Close()

			// escape recursion
			return
		}

		// convert bytes from buffer to string
		message := string(bufferBytes)
		// get the remote address of the client
		clientAddr := conn.RemoteAddr().String()
		// format a response
		response := fmt.Sprintf(message + " from " + clientAddr)

		// have server print out important information
		log.Printf("response: %v", response)

		// let the client know what happened
		mes.Echo(message)
	}
}

func (mes *multiEchoServer) hasClosed(id string) bool {
	ret := false
	<-mes.lock
	select {
	case <-mes.connectionManager[id].closed:
		ret = true
	default:
	}
	mes.lock <- true
	return ret
}

//the lock is for list, the other related stuff doesn't get locked
func (mes *multiEchoServer) Echo(msg string) {
	<-mes.lock
	for _, item := range mes.connectionManager {
		select {
		case item.msgQueue <- msg:
		default:
		}
	}
	mes.lock <- true
}
