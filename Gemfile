source 'https://rubygems.org'

git_source(:github) { |repo| "https://github.com/#{repo}.git" }

# Windows does not include zoneinfo files, so bundle the tzinfo-data gem
# and associated library.
install_if -> { RUBY_PLATFORM =~ %r!mingw|mswin|java! } do
  gem 'eventmachine', '1.2.7', platforms: :ruby
  gem 'tzinfo', '~> 1.2'
  gem 'tzinfo-data'
end

group :development do
  gem 'webrick'
  gem 'pry', '~> 0.14.1'
  gem 'pry-byebug'
end

gem 'jekyll', '~> 4.4.0'
gem 'erb'

# Use rake as a build system
gem 'rake'
gem 'dotenv'

# If you want to use GitHub Pages, remove the "gem "jekyll"" above and
# uncomment the line below. To upgrade, run `bundle update github-pages`.
# gem "github-pages", group: :jekyll_plugins

# If you have any plugins, put them here!
group :jekyll_plugins do
  gem 'jekyll-multiple-languages-plugin', github: 'keedrop/jekyll-multiple-languages-plugin'
  gem 'jekyll-environment-variables'
  gem 'jekyll-sitemap'
  gem 'jekyll-tidy'
end

# Performance-booster for watching directories on Windows
gem 'wdm', '~> 0.1.0', :install_if => Gem.win_platform?
