Githook
-------
Go program that receives GitHub Webhooks then executes scripts for repo deployment

Building
--------
To build, make sure you have the Go tools installed then run `go build`

Running
-------

`githook` takes a single argument, a rules file in json format. For example


    [
        {
            "url": "https://github.com/<username>/<reponame>",
            "branch": "refs/heads/master",
            "deployment_script": "/path/to/deployment/script.sh",
            "deployment_arguments": [ "arg1", "arg2", "arg3" ]
        }
    ]


To execute and keep the program running in the background after you log out:
`./githook -rules=rules_file.json &> output.log &`
