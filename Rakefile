# coding: utf-8
task default: :preview

require 'dotenv/load'

# based on https://github.com/avillafiorita/jekyll-rakefile/ (MIT License)

# launch compass
$compass = false
# check and warn before deploying if there are remote changes (if git is used)
$git_check = true
# commit and push after deploying
$git_autopush = false

# TODO: maybe read from ENV vars?
$deploy_dir = "keedrop@cronos.illunis.net:"
$rsync_rsh = "ssh -p 36077 -l keedrop -i #{ENV['DEPLOY_KEY']}"

#
# --- NO NEED TO TOUCH ANYTHING BELOW THIS LINE ---
#


#
# Tasks start here
#

desc 'Clean up generated site'
task :clean do
  cleanup
end


desc 'Preview on local machine (server with --auto)'
task preview: :clean do
  compass('compile') # so that we are sure sass has been compiled before we run the server
  compass('watch &')
  jekyll('serve -wl')
end
task serve: :preview


desc 'Build for deployment (but do not deploy)'
task :build, [:deployment_configuration] => :clean do |t, args|
  args.with_defaults(deployment_configuration: 'deploy')

  if rake_running then
    puts "\n\nWarning! An instance of rake seems to be running (it might not be *this* Rakefile, however).\n"
    puts "Building while running other tasks (e.g., preview), might create a website with broken links.\n\n"
    puts "Are you sure you want to continue? [Y|n]"

    ans = STDIN.gets.chomp
    exit if ans != 'Y'
  end

  compass('compile')
  jekyll("build --config _config.yml")
end


desc 'Build and deploy to remote server'
task :deploy, [:deployment_configuration] => :build do |t, args|
  args.with_defaults(deployment_configuration: 'deploy')

  if git_requires_attention("master") then
    puts "\n\nWarning! It seems that the local repository is not in sync with the remote.\n"
    puts "This could be ok if the local version is more recent than the remote repository.\n"
    puts "Deploying before committing might cause a regression of the website (at this or the next deploy).\n\n"
    puts "Are you sure you want to continue? [Y|n]"

    ans = STDIN.gets.chomp
    exit if ans != 'Y'
  end

  sh "rsync -av -e '#{$rsync_rsh}' --delete _site/ #{$deploy_dir}"
  time = Time.new
  File.open("_last_deploy.txt", 'w') {|f| f.write(time) }
  %x{git add -A && git commit -m "autopush by Rakefile at #{time}" && git push} if $git_autopush
end


#
# General support functions
#

# remove generated site
def cleanup
  sh 'rm -rf _site'
  compass('clean')
end

# launch jekyll
def jekyll(directives = '')
  sh 'jekyll ' + directives
end

# launch compass
def compass(command = 'compile')
  (sh 'compass ' + command) if $compass
end

# check if there is another rake task running (in addition to this one!)
def rake_running
  `ps | grep 'rake' | grep -v 'grep' | wc -l`.to_i > 1
end

def git_local_diffs
  %x{git diff --name-only} != ""
end

def git_remote_diffs branch
  %x{git fetch}
  %x{git rev-parse #{branch}} != %x{git rev-parse origin/#{branch}}
end

def git_repo?
  %x{git status} != ""
end

def git_requires_attention branch
  $git_check and git_repo? and git_remote_diffs(branch)
end
