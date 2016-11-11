FROM ubuntu:16.04

ADD bin/voyager-rackhd-service voyager-rackhd-service

CMD ./voyager-rackhd-service
