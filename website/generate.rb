require "json"
require "yaml"
require "digest"
require "fileutils"

# remove any previous files
system("mkdir -p content/artists")
system("rm -rf content/artists/*")

# load in plays by artist
raw_data = File.readlines("enriched-backup-latest.json")
play_data = raw_data.map { |l| JSON.parse(l) }
plays_grouped_by_artist = play_data.group_by { |play| play["artist"] }

# generate artist files
plays_grouped_by_artist.each do |artist_name, plays|
  plays = plays.map { |p| Hash[*p.to_a.map { |k, v| [k.split('_').collect(&:capitalize).join, v] }.flatten] }
  data = { "title" => artist_name, "plays" => plays.sort_by { |p| p["timestamp"]} }
  filename = Digest::MD5.hexdigest(artist_name)
  content = "#{data.to_yaml}---"

  File.write("content/artists/#{filename}.md", content)
end
