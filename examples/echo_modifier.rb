#!/usr/bin/env ruby
# encoding: utf-8
while data = STDIN.gets
    next unless data
    data = data.chomp

    decoded = [data].pack("H*")
    encoded = decoded.unpack("H*").first

    STDOUT.puts encoded


    STDERR.puts "[DEBUG] Original data: #{data}"
    STDERR.puts "[DEBUG] Decoded request: #{decoded}"
    STDERR.puts "[DEBUG] Encoded data: #{encoded}"
end