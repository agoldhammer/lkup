# lkup - a program to analyze nginx log files

lkup can read standard log files produced by nginx.

## Usage

lkup -a reads from stdin
<<<<<<< HEAD

lkup filename.log reads the named file

=======
lkup filename.log reads the named file
>>>>>>> 93eeb49d000b2772ecf3db607c706fb03d0372da
lkup -v prints the version and exits

### Example

ssh myserver cat /var/log/nginx/access.log | lkup -a

will look up hostnames of ip addresses and display access times and actions grouped by ip address
and sorted by access time:

>+++++++++
>---->  102.165.54.148

>*Hostname: 102.165.54.148

>*Country Code: ZA

> *South Africa Gauteng Johannesburg

> * (Lat/Long -26.199169158935547 28.0563907623291) Metro: 0

>No security flags

>

>....

>*: {IP:102.165.54.148 Time:2021-03-03 11:41:53 +0000 UTC Text:GET / HTTP/1.0}

>

>+++++++++

>---->  70.79.104.209

>*Hostname: S0106688f2ecdf253.vc.shawcable.net

>*Country Code: CA

>*Canada British Columbia North Vancouver

>* (Lat/Long 49.29270935058594 -123.04773712158203) Metro: 0

>No security flags

>....

>*: {IP:70.79.104.209 Time:2021-03-03 11:46:37 +0000 UTC Text:GET /.env HTTP/1.1}

>*: {IP:70.79.104.209 Time:2021-03-03 11:46:37 +0000 UTC Text:POST / HTTP/1.1}

>*: {IP:70.79.104.209 Time:2021-03-03 11:46:40 +0000 UTC Text:GET /.env HTTP/1.1}

>*: {IP:70.79.104.209 Time:2021-03-03 11:46:40 +0000 UTC Text:POST / HTTP/1.1}

## Configuration

ip addresses to be omitted from the analysis may be specified in a file named lkup.config,
to be placed in the .lkup subdirectory of $HOME.

Its contents should be of the form:

    \# ips to omit from output

    omit = "192.x.x.x 10.x.x.x"
