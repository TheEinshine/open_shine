import urllib.request
import urllib.error
import re

req = urllib.request.Request(
    'https://www.wsj.com/business/media/how-charlamagne-tha-god-mastered-the-modern-media-era-b0582068?mod=mw_quote_news',
    headers={'User-Agent': 'Mozilla/5.0'}
)
try:
    resp = urllib.request.urlopen(req)
    html = resp.read().decode('utf-8', errors='ignore')
except urllib.error.HTTPError as e:
    html = e.read().decode('utf-8', errors='ignore')

print("Length of HTML:", len(html))
print("Title:", re.findall(r'<meta[^>]*og:title[^>]*>', html))
print("Image:", re.findall(r'<meta[^>]*og:image[^>]*>', html))
