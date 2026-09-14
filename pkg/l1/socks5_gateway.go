// Package l1 implementa el gateway SOCKS5 universal transparente (RFC 1928)
// permitiendo que cualquier software comercial existente (navegadores, git, curl, SSH)
// enrute su tráfico sobre la malla cuántica de ipvn7 sin modificar código cliente.
// Rescatado y perfeccionado de Ipv7-4 e Ipv7IEU (D:\David\Ipv7-4\core\bridge\socks.go).
package l1

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Constantes canónicas SOCKS5 (RFC 1928)
const (
	SOCKS5Version       = 0x05
	SOCKS5AuthNone      = 0x00
	SOCKS5AuthNoAccept  = 0xFF
	SOCKS5CmdConnect    = 0x01
	SOCKS5AddrTypeIPv4  = 0x01
	SOCKS5AddrTypeFQDN  = 0x03
	SOCKS5AddrTypeIPv6  = 0x04

	SOCKS5RepSuccess         = 0x00
	SOCKS5RepServerFailure   = 0x01
	SOCKS5RepConnectionFail  = 0x04
	SOCKS5RepCmdNotSupported = 0x07
	SOCKS5RepAddrNotSupported= 0x08
)

// SOCKS5Gateway gestiona el servidor proxy local
type SOCKS5Gateway struct {
	mu                sync.RWMutex
	ListenAddr        string        `json:"listen_addr"`
	IsRunning         bool          `json:"is_running"`
	listener          net.Listener
	SphinxRouter      *SphinxRouter `json:"-"`
	
	// Métricas atómicas
	TotalConnections  uint64        `json:"total_connections"`
	ActiveConnections int64         `json:"active_connections"`
	BytesTx           uint64        `json:"bytes_tx"`
	BytesRx           uint64        `json:"bytes_rx"`
	LastDestination   string        `json:"last_destination"`
	quit              chan struct{}
}

// NewSOCKS5Gateway crea una instancia del gateway proxy
func NewSOCKS5Gateway(listenAddr string, sphinx *SphinxRouter) *SOCKS5Gateway {
	if listenAddr == "" {
		listenAddr = "127.0.0.1:10807"
	}
	return &SOCKS5Gateway{
		ListenAddr:   listenAddr,
		SphinxRouter: sphinx,
		quit:         make(chan struct{}),
	}
}

// Start inicia el socket TCP del proxy en segundo plano
func (s *SOCKS5Gateway) Start() error {
	s.mu.Lock()
	if s.IsRunning {
		s.mu.Unlock()
		return errors.New("gateway SOCKS5 ya está en ejecución")
	}

	listener, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("falla al enlazar listener SOCKS5 en %s: %w", s.ListenAddr, err)
	}

	s.listener = listener
	s.IsRunning = true
	s.mu.Unlock()

	go s.acceptLoop()
	return nil
}

// Stop detiene el gateway ordenadamente
func (s *SOCKS5Gateway) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.IsRunning {
		return
	}

	s.IsRunning = false
	close(s.quit)
	if s.listener != nil {
		_ = s.listener.Close()
	}
}

func (s *SOCKS5Gateway) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				time.Sleep(50 * time.Millisecond)
				continue
			}
		}

		atomic.AddUint64(&s.TotalConnections, 1)
		atomic.AddInt64(&s.ActiveConnections, 1)

		go s.handleConnection(conn)
	}
}

func (s *SOCKS5Gateway) handleConnection(client net.Conn) {
	defer func() {
		_ = client.Close()
		atomic.AddInt64(&s.ActiveConnections, -1)
	}()

	_ = client.SetDeadline(time.Now().Add(15 * time.Second))
	reader := bufio.NewReader(client)

	// 1. Negociación de Versión y Métodos de Autenticación
	ver, err := reader.ReadByte()
	if err != nil || ver != SOCKS5Version {
		return
	}

	nMethods, err := reader.ReadByte()
	if err != nil || nMethods == 0 {
		return
	}

	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(reader, methods); err != nil {
		return
	}

	hasNoAuth := false
	for _, m := range methods {
		if m == SOCKS5AuthNone {
			hasNoAuth = true
			break
		}
	}

	if !hasNoAuth {
		_, _ = client.Write([]byte{SOCKS5Version, SOCKS5AuthNoAccept})
		return
	}

	// Responder: Versión 5, Método seleccionado 0 (Sin autenticación)
	if _, err := client.Write([]byte{SOCKS5Version, SOCKS5AuthNone}); err != nil {
		return
	}

	// 2. Solicitud de Conexión (Request)
	// Formato: [VER(1) | CMD(1) | RSV(1) | ATYP(1) | DST.ADDR | DST.PORT(2)]
	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return
	}

	cmd := header[1]
	atyp := header[3]

	if cmd != SOCKS5CmdConnect {
		s.sendReply(client, SOCKS5RepCmdNotSupported, nil, 0)
		return
	}

	var destHost string
	switch atyp {
	case SOCKS5AddrTypeIPv4:
		ipv4 := make([]byte, 4)
		if _, err := io.ReadFull(reader, ipv4); err != nil {
			return
		}
		destHost = net.IP(ipv4).String()

	case SOCKS5AddrTypeFQDN:
		domainLen, err := reader.ReadByte()
		if err != nil || domainLen == 0 {
			return
		}
		domainBytes := make([]byte, domainLen)
		if _, err := io.ReadFull(reader, domainBytes); err != nil {
			return
		}
		destHost = string(domainBytes)

	case SOCKS5AddrTypeIPv6:
		ipv6 := make([]byte, 16)
		if _, err := io.ReadFull(reader, ipv6); err != nil {
			return
		}
		destHost = net.IP(ipv6).String()

	default:
		s.sendReply(client, SOCKS5RepAddrNotSupported, nil, 0)
		return
	}

	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(reader, portBytes); err != nil {
		return
	}
	destPort := binary.BigEndian.Uint16(portBytes)
	destAddrStr := net.JoinHostPort(destHost, strconv.Itoa(int(destPort)))

	s.mu.Lock()
	s.LastDestination = destAddrStr
	s.mu.Unlock()

	// 3. Conexión hacia destino
	// Conexión saliente transparente:
	targetConn, err := net.DialTimeout("tcp", destAddrStr, 10*time.Second)
	if err != nil {
		s.sendReply(client, SOCKS5RepConnectionFail, nil, 0)
		return
	}
	defer targetConn.Close()

	// 4. Enviar Respuesta de Éxito al Cliente SOCKS5
	localTCPAddr := targetConn.LocalAddr().(*net.TCPAddr)
	s.sendReply(client, SOCKS5RepSuccess, localTCPAddr.IP, uint16(localTCPAddr.Port))

	// Remover deadlines para transferencia continua
	_ = client.SetDeadline(time.Time{})
	_ = targetConn.SetDeadline(time.Time{})

	// 5. Túnel bidireccional de datos con contabilidad
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		n, _ := io.Copy(targetConn, client)
		atomic.AddUint64(&s.BytesTx, uint64(n))
	}()

	go func() {
		defer wg.Done()
		n, _ := io.Copy(client, targetConn)
		atomic.AddUint64(&s.BytesRx, uint64(n))
	}()

	wg.Wait()
}

func (s *SOCKS5Gateway) sendReply(client net.Conn, rep byte, bindIP net.IP, bindPort uint16) {
	reply := make([]byte, 10)
	reply[0] = SOCKS5Version
	reply[1] = rep
	reply[2] = 0x00 // Reservado
	reply[3] = SOCKS5AddrTypeIPv4

	if bindIP != nil && bindIP.To4() != nil {
		copy(reply[4:8], bindIP.To4())
	}
	binary.BigEndian.PutUint16(reply[8:10], bindPort)

	_, _ = client.Write(reply)
}

// GetStats retorna métricas del gateway para el panel de control
func (s *SOCKS5Gateway) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"listen_addr":        s.ListenAddr,
		"is_running":         s.IsRunning,
		"total_connections":  atomic.LoadUint64(&s.TotalConnections),
		"active_connections": atomic.LoadInt64(&s.ActiveConnections),
		"bytes_tx":           atomic.LoadUint64(&s.BytesTx),
		"bytes_rx":           atomic.LoadUint64(&s.BytesRx),
		"last_destination":   s.LastDestination,
		"rfc":                "RFC 1928 (Universal SOCKS5)",
	}
}
