# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "precise32"

  config.vm.box_url = "http://files.vagrantup.com/precise32.box"
  
  config.vm.provision :shell, :inline => "
  if which go >/dev/null; then
    echo 'second run'
  else
    apt-get install -y curl vim git build-essential
    curl https://go.googlecode.com/files/go1.1.linux-amd64.tar.gz > ~/go1.1.tar.gz
    cd ~/
    tar xzf go1.1.tar.gz

    echo 'export GOROOT=$HOME/go' >> ~/.profile
    echo 'export PATH=$PATH:$GOROOT/bin' >> ~/.profile
  fi
  "

  config.vm.synced_folder "~/.ssh", "/home/vagrant/.ssh"
end
