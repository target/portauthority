## Port Authority v1 API

All API examples here are to be used with [Postman](https://www.getpostman.com/) and are directed at the default Minikube cluster ip `http://192.168.99.100` and Port Authority's exposed NodePort `31700`. Curl API examples can be found [here](/docs/README.md).

### Environment Setup (Minikube)

The Postman environment variables for Minikube can be imported from the [environments folder](environments/portauthority - minikube.postman_environment.json). This defines the hostname, version, port, and Kubernetes config for the API examples to run.

*NOTE: The value for env variable kube_config needs to be populated with Minikube's flattened and* **base64** *encoded config. (`kubectl config view --flatten=true | base64 | pbcopy`)*

### Collection Setup (Minikube)

The Postman collection of API examples for Minikube can be imported from the [collections folder](collections/portauthority Examples.postman_collection.json). This covers all of the API functionality except creating registry crawlers.

*NOTE: Public registry crawling is not supported, though external registries can be crawled using the following example.*

#### Example Registry Request

```
curl -X POST \
  http://192.168.99.100:31700/v1/crawlers/registry  \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  -d '{
      "RegCrawler":
      {
        "Registry": "mybinrepo",
        "Repos": ["path/toimage"],
        "Tags": ["latest"],
        "MaxThreads": 100,
        "Username": "mybinrepo_username",
        "Password": "mybinrepo_password"
      }
    }'
```

#### Example Response

```json
{
  "Crawler": {
    "ID": 2,
    "Type": "registry",
    "Started": "2017-08-11T17:03:32.845608Z",
    "Finished": "0000-00-00T00:00:00.000000Z"
  }
}
```
