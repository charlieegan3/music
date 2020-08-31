#!/usr/bin/env ruby

require "json"
require "yaml"
require "digest"
require "fileutils"
require "open-uri"

def run(command)
  puts "running: #{command}"
  fail unless system(command)
end

BACKUP_FILE = "enriched-backup-latest.json"
BACKUP_LOCATION = "https://storage.googleapis.com/charlieegan3-music-backup/#{BACKUP_FILE}"
HUGO_RELEASE = "https://github.com/gohugoio/hugo/releases/download/v0.69.2/hugo_0.69.2_Linux-64bit.tar.gz"

# install hugo if missing
unless File.exists?("hugo")
  puts "hugo missing, installing"
  run("curl -L #{HUGO_RELEASE} > hugo.tar.gz")
  run("tar -zxf hugo.tar.gz")
end

# download the play data
unless File.exists?(BACKUP_FILE)
  puts "downloading play data"
  File.write(BACKUP_FILE, open(BACKUP_LOCATION).read)
end

# remove any previous files
run("mkdir -p content/artists")
run("rm -r content/artists/*")

# load in plays by artist
puts "loading play data"
raw_data = File.readlines("enriched-backup-latest.json")
play_data = raw_data.map { |l| JSON.parse(l) }
plays_grouped_by_artist = play_data.group_by { |play| play["artist"] }

# generate artist files
puts "generate hugo site source"
total = plays_grouped_by_artist.size
count = 0
plays_grouped_by_artist.each do |artist_name, plays|
  plays = plays.map { |p| Hash[*p.to_a.map { |k, v| [k.split('_').collect(&:capitalize).join, v] }.flatten] }
  data = { "title" => artist_name, "plays" => plays.sort_by { |p| p["timestamp"]} }
  filename = Digest::MD5.hexdigest(artist_name)
  content = "#{data.to_yaml}---"

  File.write("content/artists/#{filename}.md", content)

  count +=1
  if count % 500 == 0
    print "#{((count.to_f/total) * 100).round}%\r"
  end
end
puts

# build the site and move to docs
puts "build hugo site"
run("./hugo")

# commit the result
email = `git config --global user.email`.chomp
name = `git config --global user.name`.chomp
if name == "" || email == ""
  puts "setting gh actions git identity"
  run('git config --global user.email "githubactions@example.com"')
  run('git config --global user.name "GitHub Actions"')
end
run("git checkout -b netlify")
run("git add public")
run("git -c commit.gpgsign=false commit -m generate-site")
run("git push -f origin netlify")
