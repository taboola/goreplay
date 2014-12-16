#!/usr/bin/env ruby
# encoding: utf-8
require "base64"

STDERR.puts "Starting modifier"
puts "Starting modifier"

while data = STDIN.gets.chomp
  STDERR.puts "==== Start ===="
  STDERR.puts Base64.encode64(data)
  puts data
  STDERR.puts "==== End ===="
end