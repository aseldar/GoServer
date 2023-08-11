package main

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/sync/semaphore"
)

func main() {
	// Создаем семафор с максимальным весом maxConcurrentConnections.
	maxConcurrentConnections := int64(10) // Пример значения
	sem := semaphore.NewWeighted(maxConcurrentConnections)
	var totalBytes int64

	// Создаем файл лога.
	logFile, err := os.OpenFile("network.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	defer logFile.Close()

	// Устанавливаем вывод логера по умолчанию в файл лога.
	log.SetOutput(logFile)

	// Прослушиваем входящие соединения.
	l, err := net.Listen("tcp", ":2000")
	if err != nil {
		log.Fatalf("Error listening: %v", err)
	}

	// Закрываем слушателя при закрытии приложения.
	defer func() {
		if err = l.Close(); err != nil {
			log.Printf("Error closing listener: %v", err)
		}
	}()
	log.Println("Listening on localhost:2000")

	// ...
	for {
		// Прослушиваем входящее соединение.
		conn, err := l.Accept()
		if err != nil {
			log.Printf("Error accepting: %v", err)
			continue
		}
		// Устанавливаем KeepAlive для соединения.
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(5 * time.Minute)
		}

		// Пытаемся получить семафор. Если горутины maxConcurrentConnections уже запущены, это заблокирует их до тех пор,
		// пока одна из них не завершится и не выдаст семафор.
		if err := sem.Acquire(context.Background(), 1); err != nil {
			log.Printf("Failed to acquire semaphore: %v", err)
			continue
		}

		// Обрабатываем соединения в новой горутине.
		go func() {
			// Выпускаем семафор при завершении горутины.
			defer sem.Release(1)
			handleRequest(conn, &totalBytes)
		}()
	}
}

func handleRequest(conn net.Conn, totalBytes *int64) {
	// Создаем буфер для хранения входящих данных.
	buf := make([]byte, 1024)
	for {
		// Записываем входящее соединение в буфер.
		len, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				log.Println("Connection closed by the client")
			} else {
				log.Printf("Error reading: %v", err)
			}
			break
		}
		// Занесим в лог в IP-адрес и порт удаленного клиента.
		log.Printf("Received connection from: %s", conn.RemoteAddr().String())

		// Заносим в лог время соединения
		log.Printf("Time of connection: %s", time.Now().Format(time.RFC3339))

		// Добавляем к общему количеству полученных байтов и заносим новую сумму в лог
		*totalBytes += int64(len) // Увеличиваем значение, на которое указывает указатель
		log.Printf("Received %d bytes from the client. Total bytes received: %d", len, *totalBytes)

		// Заносим в лог входящее сообщение.
		log.Printf("Received message: %s", string(buf[:len]))
	}

	// Закрываем соединение после его обработки.
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}()
}
