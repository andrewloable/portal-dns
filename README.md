# portal-dns
Customized DNS Server for portal access

## Purpose
1. Conditional DNS access, forward the requests to a corporate IP address if the device or user is not authenticated.
2. If device or user is authenticated, forward and/or cache the resolved addressed from external DNS Server

## Compile
1. ```git clone git clone git@github.com:andrewloable/portal-dns.git```
2. Go to the cloned directory
3. ```go build```
4. ```sudo ./portal-dns```
