// Package netconfcore 是自研 NETCONF 客户端的协议核心（替代 scrapligo 计划 Wave 1）。
//
// 本文件实现 RFC 6242 的两种报文封帧：
//   - EOM（NETCONF 1.0）：帧以 ]]>]]> 定界，hello 阶段恒用此格式；
//   - chunked（NETCONF 1.1）：\n#<size>\n<data>… 以 \n##\n 结束，能力协商后启用。
//
// 设计约束：零第三方依赖（R10）；单实例供单协程使用（会话级并发控制在上层
// RPC 引擎做，实例间零共享状态）；所有读路径设总量上限防异常设备打爆内存（R08）。
package netconfcore

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

const eomDelimiter = "]]>]]>"

// maxFrameSize 单帧（含多 chunk 重组后）上限。全量配置回读在数 MB 量级，
// 64MB 给足余量；超限视为协议异常切断，防 OOM。变量而非常量：测试注入小上限
// 以毫秒级验证切断路径（生产代码不得改写）。chunk-size 上限同帧上限
// （RFC 6242 名义上限 4294967295，收紧即可）。
var maxFrameSize = 64 << 20

// FrameWriter 把一条完整 NETCONF 消息封帧写出。
type FrameWriter interface {
	WriteFrame(msg []byte) error
}

// FrameReader 读出一条完整 NETCONF 消息（已去封帧）。
type FrameReader interface {
	ReadFrame() ([]byte, error)
}

// ── EOM ─────────────────────────────────────────────────────────

type eomWriter struct{ w io.Writer }

// NewEOMWriter 返回 NETCONF 1.0 EOM 封帧写端。
func NewEOMWriter(w io.Writer) FrameWriter { return &eomWriter{w: w} }

func (e *eomWriter) WriteFrame(msg []byte) error {
	if _, err := e.w.Write(msg); err != nil {
		return fmt.Errorf("netconfcore: 写帧: %w", err)
	}
	// 定界符后补 \n：RFC 语义无影响，但按行读流的服务端（netconfsim 及部分
	// 真机实现）依赖它完成收包；scrapligo 同款行为。
	if _, err := io.WriteString(e.w, eomDelimiter+"\n"); err != nil {
		return fmt.Errorf("netconfcore: 写定界符: %w", err)
	}
	return nil
}

type eomReader struct{ r *bufio.Reader }

// asBufio 复用已有 bufio.Reader，避免多层缓冲吞字节：会话 hello 阶段用 EOM
// 读端、协商后切 chunked 读端，两者必须共享同一缓冲才不丢已预读的数据。
func asBufio(r io.Reader) *bufio.Reader {
	if br, ok := r.(*bufio.Reader); ok {
		return br
	}
	return bufio.NewReader(r)
}

// NewEOMReader 返回 NETCONF 1.0 EOM 切帧读端。
func NewEOMReader(r io.Reader) FrameReader {
	return &eomReader{r: asBufio(r)}
}

func (e *eomReader) ReadFrame() ([]byte, error) {
	// 逐字节累积并检查缓冲尾部是否恰为定界符。相比前缀状态机，天然规避
	// 载荷尾部 ]…] 与定界符首字符粘连时的回退陷阱（KMP 失配函数问题），
	// 每字节仅比对尾部 6 字节，代价可忽略。
	var buf bytes.Buffer
	delim := []byte(eomDelimiter)
	for {
		b, err := e.r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if buf.Len() == 0 {
					return nil, io.EOF
				}
				return nil, io.ErrUnexpectedEOF
			}
			return nil, fmt.Errorf("netconfcore: 读帧: %w", err)
		}
		// 跳过帧首空白：对端可能在定界符后附加 \r\n（本写端亦如此），
		// 残留会粘到下一帧开头；NETCONF 消息是 XML，帧首空白无语义。
		if buf.Len() == 0 && (b == '\n' || b == '\r') {
			continue
		}
		buf.WriteByte(b)
		if buf.Len() >= len(delim) && bytes.HasSuffix(buf.Bytes(), delim) {
			frame := make([]byte, buf.Len()-len(delim))
			copy(frame, buf.Bytes())
			return frame, nil
		}
		if buf.Len() > maxFrameSize {
			return nil, fmt.Errorf("netconfcore: 帧超上限 %d 字节，切断（防异常设备）", maxFrameSize)
		}
	}
}

// ── Chunked（RFC 6242 §4.2）─────────────────────────────────────

type chunkedWriter struct{ w io.Writer }

// NewChunkedWriter 返回 NETCONF 1.1 chunked 封帧写端（整帧作单 chunk，RFC 允许）。
func NewChunkedWriter(w io.Writer) FrameWriter { return &chunkedWriter{w: w} }

func (c *chunkedWriter) WriteFrame(msg []byte) error {
	if len(msg) == 0 {
		return errors.New("netconfcore: 拒绝写空帧（RFC6242 chunk-size ≥1）")
	}
	if _, err := fmt.Fprintf(c.w, "\n#%d\n", len(msg)); err != nil {
		return fmt.Errorf("netconfcore: 写 chunk 头: %w", err)
	}
	if _, err := c.w.Write(msg); err != nil {
		return fmt.Errorf("netconfcore: 写 chunk 数据: %w", err)
	}
	if _, err := io.WriteString(c.w, "\n##\n"); err != nil {
		return fmt.Errorf("netconfcore: 写结束标记: %w", err)
	}
	return nil
}

type chunkedReader struct{ r *bufio.Reader }

// NewChunkedReader 返回 NETCONF 1.1 chunked 切帧读端（多 chunk 自动重组）。
func NewChunkedReader(r io.Reader) FrameReader {
	return &chunkedReader{r: asBufio(r)}
}

func (c *chunkedReader) ReadFrame() ([]byte, error) {
	var buf bytes.Buffer
	first := true
	for {
		size, last, err := c.readChunkHeader(first)
		if err != nil {
			if first && errors.Is(err, io.EOF) {
				return nil, io.EOF
			}
			// chunk 级观测（真机排障）：读帧失败时打出已累计字节——
			// 「设备慢」与「解析卡死在半帧」在完整帧日志里都隐形，靠这行区分。
			wireLogf("chunked read error after partial %dB: %v", buf.Len(), err)
			return nil, err
		}
		first = false
		if last {
			if buf.Len() == 0 {
				return nil, errors.New("netconfcore: 空帧（无任何 chunk 即结束）")
			}
			return buf.Bytes(), nil
		}
		if buf.Len()+size > maxFrameSize {
			return nil, fmt.Errorf("netconfcore: 帧累计超上限 %d 字节，切断（防异常设备）", maxFrameSize)
		}
		if _, err := io.CopyN(&buf, c.r, int64(size)); err != nil {
			wireLogf("chunked read error after partial %dB: chunk 数据不足", buf.Len())
			return nil, fmt.Errorf("netconfcore: chunk 数据不足 %d 字节: %w", size, io.ErrUnexpectedEOF)
		}
		wireLogf("chunk %dB (total %dB)", size, buf.Len())
	}
}

// readChunkHeader 解析 \n#<size>\n；\n##\n 时返回 last=true。
// 前导空白宽容：帧间可能残留对端附加的 \r\n（EOM hello 切 chunked 的过渡、
// 部分实现的帧尾换行），跳过零或多个后必须见 '#'。
func (c *chunkedReader) readChunkHeader(first bool) (size int, last bool, err error) {
	for {
		b, err := c.r.ReadByte()
		if err != nil {
			if first && errors.Is(err, io.EOF) {
				return 0, false, io.EOF
			}
			return 0, false, fmt.Errorf("netconfcore: chunk 头断流: %w", io.ErrUnexpectedEOF)
		}
		if b == '\n' || b == '\r' {
			continue
		}
		if b != '#' {
			return 0, false, fmt.Errorf("netconfcore: chunk 头非法（期望 '#' 读到 %q）", b)
		}
		break
	}
	peek, err := c.r.ReadByte()
	if err != nil {
		return 0, false, fmt.Errorf("netconfcore: chunk 头断流: %w", io.ErrUnexpectedEOF)
	}
	if peek == '#' { // \n##\n 结束标记
		b, err := c.r.ReadByte()
		if err != nil || b != '\n' {
			return 0, false, errors.New("netconfcore: 结束标记残缺（期望 \\n## 后接 \\n）")
		}
		return 0, true, nil
	}
	// 收集十进制 chunk-size（首位非零，RFC 6242）
	digits := []byte{peek}
	for {
		b, err := c.r.ReadByte()
		if err != nil {
			return 0, false, fmt.Errorf("netconfcore: chunk 头断流: %w", io.ErrUnexpectedEOF)
		}
		if b == '\n' {
			break
		}
		digits = append(digits, b)
		if len(digits) > 10 {
			return 0, false, errors.New("netconfcore: chunk-size 位数超 RFC 上限")
		}
	}
	if digits[0] < '1' || digits[0] > '9' {
		return 0, false, fmt.Errorf("netconfcore: chunk-size 非法: %q", digits)
	}
	n, convErr := strconv.Atoi(string(digits))
	if convErr != nil {
		return 0, false, fmt.Errorf("netconfcore: chunk-size 非数字: %q", digits)
	}
	if n > maxFrameSize {
		return 0, false, fmt.Errorf("netconfcore: chunk-size %d 超上限", n)
	}
	return n, false, nil
}
