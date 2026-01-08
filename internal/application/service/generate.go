package service

//go:generate go run ../../tools/mocks_helper.go
//go:generate minimock -i CrawlJobService -o ./mocks/ -s "_minimock.go"
//go:generate minimock -i CrawlTaskService -o ./mocks/ -s "_minimock.go"
