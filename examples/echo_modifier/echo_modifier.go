package main

import (
    "os"
    "bufio"
)

func main() {
    reader := bufio.NewReader(os.Stdin)
    data := make(chan []byte)

    go ReadStdin(data)

    for {
        os.Stdout.Print(<- data, '¶')
    }
}

func ReadStdin(data chan []byte){
    for {
        buf, err := reader.ReadBytes('¶')
        buf_len := len(buf)
        if buf_len > 0 {
            new_buf_len := len(buf) - 2
            if new_buf_len > 0 {
                new_buf := make([]byte, new_buf_len)
                copy(new_buf, buf[:new_buf_len])
                data <- new_buf
                if err != nil {
                    if err != io.EOF {
                        log.Printf("error: %s\n", err)
                    }
                }
            }
        }
    }
}

while data = STDIN.gets(separator)
  STDERR.puts "==== Start ===="
  STDERR.puts data
  puts data
  STDERR.puts "==== End ===="
end