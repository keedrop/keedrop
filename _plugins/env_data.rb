# Fill in content from environment variables
# JEKYLL_DATA_CONTACT_NAME becomes site.data.contact.name
class EnvData < Jekyll::Generator

  CONTENT_PREFIX = "JEKYLL_DATA_".freeze

  def generate(site)
    ENV.each do |key, value|
      next unless key.start_with?(CONTENT_PREFIX)
      fill_env(site.data, key[CONTENT_PREFIX.length..-1].downcase.split('_'), value)
    end
  end

  private

  def fill_env(store, names, value)
    if names.length > 1
      nested = names.slice!(0)
      store[nested] ||= OpenStruct.new
      fill_env(store[nested], names, value)
    else
      store[names.first] = value
    end
  end
end
