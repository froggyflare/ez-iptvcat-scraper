import pycurl
import urllib.request
import json
from io import BytesIO, StringIO

ALL_STREAMS_JSON_LINK = 'https://raw.githubusercontent.com/froggyflare/ez-iptvcat-scraper/master/data/all-streams.json'
M3U_OUTPUT = './data/all-stream.m3u8'

# Download Json Data File
response = urllib.request.urlopen(ALL_STREAMS_JSON_LINK)
if response.status != 200:
    print ("Couldn't get a good response:" + response.read())
data = response.read()
data_decoded = data.decode('utf-8')
json_streams = json.loads(data_decoded)

result_file = open(M3U_OUTPUT, 'w')
# For each download M3u and merge
for stream in json_streams:
    stream_link = stream['link']
    print("Link:" + stream_link)
    headers = {}

    crl = pycurl.Curl()
    resp = BytesIO()
    crl.setopt(crl.URL, stream_link)
    crl.setopt(crl.WRITEDATA, resp)
    crl.perform()
    crl.close()

    m3u_segment = resp.getvalue().decode('UTF-8')
    print("segment:" + m3u_segment)
    result_file.write(m3u_segment)

result_file.close()
