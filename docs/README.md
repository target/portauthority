### Port Authority v1 API

All API examples are curl commands directed at the default Minikube cluster ip `http://192.168.99.100` and port authority's exposed NodePort `31700`. [Postman](https://www.getpostman.com/) API examples can be found [here](postman/README.md)

- [Images](#images)
  - [POST](#post-images)
  - [GET](#get-imagesid)
  - [LIST](#list-images)
- [Containers](#containers)
  - [GET](#get-containersid)
  - [LIST](#list-containers)
- [Policies](#policies)
  - [POST](#post-policies)
  - [GET](#get-policiesname)
  - [LIST](#list-policies)
- [Crawlers](#crawlers)
  - [POST](#post-crawlerstype)
  - [GET](#get-crawlersid)
- [K8s-Image-Policy-Webhook](#k8s-image-policy-webhook)
  - [POST](#post-k8s-image-policy-webhook)

## Images

### POST /images

#### Description

The POST route for the Image resource performs the analysis of an Image from the given registry repo and tag name parameters.

The Docker manifest is pulled from the registry and the proper layer path is sent to [Clair Layer API](https://github.com/coreos/clair/blob/master/Documentation/api_v1.md#post-layers) in the correct order.  The layer parent and child names are hashed with the image content digest to provide a unique name per layer so that the proper order can be maintained within the Clair database.

When no registry credentials are available configuration will be checked for appropriate ones.

#### Example Request

```
curl -X POST \
  http://192.168.99.100:31700/v1/images \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  -d '{
  "Image": {
    "Registry": "https://registry-1.docker.io",
    "Repo": "library/postgres",
    "Tag": "latest",
    "RegistryUser": "",
    "RegistryPassword": "",
    "Metadata": {
      "data": "is so meta"
    }
	}
}'
```

#### Example Response

```json
{
    "Image": {
        "ID": 10,
        "Registry": "https://registry-1.docker.io",
        "Repo": "library/postgres",
        "Tag": "latest",
        "Digest": "sha256:d5787305ec0a3b9a24d0108cb5fdbb4befbd809f85639bf04aa1941138df9701",
        "FirstSeen": "2018-04-02T16:18:22.632404Z",
        "LastSeen": "2018-04-02T17:11:17.515506Z",
        "Metadata": {
            "data": "is so meta"
        }
    }
}
```

### GET /images/`:id`

#### Description

The GET route for the Image resource displays an Image and optionally all of its features, vulnerabilities and policy violations. You will need to use ID given from the POST image response like in the example above.

#### Query Parameters

| Name            | Type   | Required | Description                                                                     |
|-----------------|--------|----------|---------------------------------------------------------------------------------|
| features        | bool   | optional | Displays the list of features indexed in this layer and all of its parents.     |
| vulnerabilities | bool   | optional | Displays the list of vulnerabilities along with the features described above.   |
| policy          | string | optional | Displays the list of vulnerabilities that have violated a defined policy. |

#### Example Request

```
curl \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  http://192.168.99.100:31700/v1/images/1?vulnerabilities&policy=default
```

#### Example Response

```json
{
  "Image": {
    "ID": 1,
    "Registry": "https://registry-1.docker.io",
    "Repo": "library/postgres",
    "Tag": "9.6",
    "Digest": "sha256:8df3344385deb0d5d781c442b2e275f7c321d601652d6317ce25ed8ffad03427",
    "FirstSeen": "2018-04-04T18:47:49.592846Z",
    "LastSeen": "2018-04-04T18:47:49.592846Z",
    "Features": [
      {
        "Name": "libc-utils",
        "NamespaceName": "alpine:v3.5",
        "VersionFormat": "dpkg",
        "Version": "0.7-r1",
        "AddedBy": "c308fd12aa2ed79c65520d7d8c69169c"
      },
      {
        "Name": "zlib",
        "NamespaceName": "alpine:v3.5",
        "VersionFormat": "dpkg",
        "Version": "1.2.8-r2",
        "Vulnerabilities": [
          {
            "Name": "CVE-2016-9843",
            "NamespaceName": "alpine:v3.5",
            "Link": "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-9843",
            "Severity": "Unknown",
            "FixedBy": "1.2.11-r0"
          }
        ],
        "AddedBy": "c308fd12aa2ed79c65520d7d8c69169c"
      }
    ],
    "Violations": [
      {
        "Type": "Basic",
        "FeatureName": "zlib",
        "FeatureVersion": "1.2.8-r2",
        "Vulnerability": {
          "Name": "CVE-2016-9843",
          "NamespaceName": "alpine:v3.5",
          "Link": "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-9843",
          "Severity": "Unknown",
          "FixedBy": "1.2.11-r0"
        }
      }
    ],
    "Metadata": {
      "data": "is so meta"
    }
  }
}
```

### LIST /images

#### Description

The LIST route for the Image resource displays all Images with optional query parameters.

#### Query Parameters

| Name            | Type   | Required | Description                                             |
|-----------------|--------|----------|---------------------------------------------------------|
| registry        | string | optional | Shows only images from specific registry.               |
| repo            | string | optional | Shows only images from specific repo.                   |
| tag             | string | optional | Shows only images with a specific tag.                  |
| digest          | string | optional | Shows only images with a specific digest.               |
| date_start      | string | optional | Shows only images last seen after specified start date. |
| date_end        | string | optional | Shows only images last seen before specified end date.  |
| limit           | string | optional | Limits quantity of images shown.                        |

#### Example Request

```
curl \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  http://192.168.99.100:31700/v1/images?date_start=2018-04-03&limit=2
```

#### Example Response

```json
{
    "Images": [
        {
            "ID": 1,
            "Registry": "https://registry-1.docker.io",
            "Repo": "library/postgres",
            "Tag": "9.6",
            "Digest": "sha256:8df3344385deb0d5d781c442b2e275f7c321d601652d6317ce25ed8ffad03427",
            "FirstSeen": "2018-04-04T18:47:49.592846Z",
            "LastSeen": "2018-04-04T18:47:49.592846Z",
            "Metadata": {
                "data": "is so meta"
            }
        },
        {
            "ID": 2,
            "Registry": "https://registry-1.docker.io",
            "Repo": "arminc/clair-db",
            "Tag": "latest",
            "Digest": "sha256:cebd94b52087407fae5ebb902f2595bdacb1357e81d00db92b70a73cda93ed3b",
            "FirstSeen": "2018-04-04T18:47:49.741407Z",
            "LastSeen": "2018-04-04T18:47:49.741408Z",
            "Metadata": {
                "data": "is so meta"
            }
        }
    ]
}
```

## Containers

### GET /containers/`:id`

#### Description

The GET route for a container with a specific id provides a deep view with all of the associated Image resources and optionally all of their features, vulnerabilities and policy violations. You will need to use ID given from the GET list container route shown below.

#### Query Parameters

| Name            | Type   | Required | Description                                                                                            |
|-----------------|--------|----------|--------------------------------------------------------------------------------------------------------|
| features        | bool   | optional | Displays the list of features indexed in this layer and all of its parents for all container images.   |
| vulnerabilities | bool   | optional | Displays the list of vulnerabilities along with the features described above for all container images. |
| policy          | string | optional | Displays the list of vulnerabilities that have violated a defined policy for all container images.     |

#### Example Request

```
curl \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  http://192.168.99.100:31700/v1/containers/5?features&vulnerabilities&policy=default
```

#### Example Response

```json
{
  "Container": {
    "ID": 5,
    "Namespace": "default",
    "Cluster": "https://192.168.99.100:8443",
    "Name": "postgres",
    "Image": "postgres:9.6",
    "ImageScanned": true,
    "ImageID": "docker-pullable://postgres@sha256:eda798e53a1a2684308c3d6408400c39e1431892e77d7d790d65d64a14467a43",
    "ImageRegistry": "",
    "ImageRepo": "postgres",
    "ImageTag": "9.6",
    "ImageDigest": "sha256:eda798e53a1a2684308c3d6408400c39e1431892e77d7d790d65d64a14467a43",
    "Annotations": {
        "kubernetes": "annotation"
    },
    "FirstSeen": "2018-04-04T18:47:47.269509Z",
    "LastSeen": "2018-04-04T18:47:47.269509Z",
    "Features": [
      {
        "Name": "libc-utils",
        "NamespaceName": "alpine:v3.5",
        "VersionFormat": "dpkg",
        "Version": "0.7-r1",
        "AddedBy": "c308fd12aa2ed79c65520d7d8c69169c"
      },
      {
        "Name": "zlib",
        "NamespaceName": "alpine:v3.5",
        "VersionFormat": "dpkg",
        "Version": "1.2.8-r2",
        "Vulnerabilities": [
          {
            "Name": "CVE-2016-9843",
            "NamespaceName": "alpine:v3.5",
            "Link": "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-9843",
            "Severity": "Unknown",
            "FixedBy": "1.2.11-r0"
          }
        ],
        "AddedBy": "c308fd12aa2ed79c65520d7d8c69169c"
      }
    ],
    "Violations": [
      {
        "Type": "Basic",
        "FeatureName": "zlib",
        "FeatureVersion": "1.2.8-r2",
        "Vulnerability": {
          "Name": "CVE-2016-9843",
          "NamespaceName": "alpine:v3.5",
          "Link": "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-9843",
          "Severity": "Unknown",
          "FixedBy": "1.2.11-r0"
        }
      }
    ]
  }
}
```

### LIST /containers

#### Description

The LIST route for the containers resource lists all containers with optional query parameters.

#### Query Parameters

| Name            | Type   | Required | Description                                                 |
|-----------------|--------|----------|-------------------------------------------------------------|
| namespace       | string | optional | Shows only containers from a specific namespace.            |
| cluster         | string | optional | Shows only containers from a specific cluster.              |
| name            | string | optional | Shows only containers with a specific name.                 |
| image           | string | optional | Shows only containers with a specific image.                |
| image_id        | string | optional | Shows only containers with a specific image_id.             |
| date_start      | string | optional | Shows only containers last seen after specified start date. |
| date_end        | string | optional | Shows only containers last seen before specified end date.  |
| limit           | string | optional | Limits quantity of containers shown.                        |

#### Example Request

```
curl \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  http://192.168.99.100:31700/v1/containers?namespace=default&limit=2
```

#### Example Response

```json
{
  "Containers": [
    {
        "ID": 4,
        "Namespace": "default",
        "Cluster": "https://192.168.99.100:8443",
        "Name": "portauthority",
        "Image": "portauthority:latest",
        "ImageID": "docker://sha256:0b6ba7e5320175965a557cd74c418f15e897ec145818f96a5bcd76c8b8050e6a",
        "ImageRegistry": "",
        "ImageRepo": "portauthority",
        "ImageTag": "latest",
        "ImageDigest": "sha256:0b6ba7e5320175965a557cd74c418f15e897ec145818f96a5bcd76c8b8050e6a",
        "Annotations": {
            "kubernetes": "annotation"
        },
        "FirstSeen": "2018-04-04T18:47:47.266826Z",
        "LastSeen": "2018-04-04T18:47:47.266826Z"
    },
    {
        "ID": 5,
        "Namespace": "default",
        "Cluster": "https://192.168.99.100:8443",
        "Name": "postgres",
        "Image": "postgres:9.6",
        "ImageID": "docker-pullable://postgres@sha256:eda798e53a1a2684308c3d6408400c39e1431892e77d7d790d65d64a14467a43",
        "ImageRegistry": "",
        "ImageRepo": "postgres",
        "ImageTag": "9.6",
        "ImageDigest": "sha256:eda798e53a1a2684308c3d6408400c39e1431892e77d7d790d65d64a14467a43",
        "ApplicationID": "",
        "FirstSeen": "2018-04-04T18:47:47.269509Z",
        "LastSeen": "2018-04-04T18:47:47.269509Z"
    }
  ]
}
```

## Policies

### POST /policies

#### Description

The POST route for the Policy creates a policy that can be set against an image with vulnerabilities. The policy will create a list of Violations based on its configuration.

A policy named `default` is created when first initializing the Port Authority database.

#### Example Request

```
curl -X POST \
  http://192.168.99.100:31700/v1/policies \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  -d '{
    "Policy": {
      "Name": "default",
      "AllowedRiskSeverity": "",
      "AllowedCVENames": "",
      "AllowNotFixed": false,
      "NotAllowedCveNames": "",
      "NotAllowedOSNames": ""
    }
  }'
```

#### Example Response

```json
{
    "Policy": {
        "ID": 1,
        "Name": "default",
        "AllowedRiskSeverity": "[]",
        "AllowedCVENames": "[]",
        "NotAllowedCveNames": "[]",
        "NotAllowedOSNames": "[]",
        "Created": "2017-11-29T18:04:52.501915Z",
        "Updated": "2017-11-29T19:28:52.657861Z"
    }
}
```
### LIST /policies

#### Description

The LIST route for the Policy resource lists all Policies

#### Example Request

```
curl \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  http://192.168.99.100:31700/v1/policies
```

#### Example Response

```json
{
    "Policies": [
        {
            "ID": 1,
            "Name": "default",
            "AllowedRiskSeverity": "[]",
            "AllowedCVENames": "[]",
            "AllowNotFixed": false,
            "NotAllowedCveNames": "[]",
            "NotAllowedOSNames": "[]",
            "Created": "2018-04-04T18:46:17.394311Z",
            "Updated": "2018-04-04T18:56:56.225075Z"
        },
        {
            "ID": 2,
            "Name": "low",
            "AllowedRiskSeverity": "[]",
            "AllowedCVENames": "[]",
            "AllowNotFixed": true,
            "NotAllowedCveNames": "[]",
            "NotAllowedOSNames": "[]",
            "Created": "2018-04-04T18:46:17.394311Z",
            "Updated": "2018-04-04T18:56:56.225075Z"
        }
    ]
}
```

### GET /policies/`:name`

#### Description

The GET route for the Policy resource displays an Policy

#### Example Request

```
curl \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  http://192.168.99.100:31700/v1/policies/default
```

#### Example Response

```json
{
    "Policy": {
        "ID": 8,
        "Name": "default",
        "AllowedRiskSeverity": "[]",
        "AllowedCVENames": "[]",
        "NotAllowedCveNames": "[]",
        "NotAllowedOSNames": "[]",
        "Created": "2017-11-29T18:04:52.501915Z",
        "Updated": "2017-11-29T19:28:52.657861Z"
    }
}
```

## Crawlers

### POST /crawlers/`:type`

#### Description

The POST route for the Crawlers resource performs one of two actions based on the `type` string parameter. Once initialized the crawl action occurs the actions run in the background on the Port Authority Server. The status of the crawler and any error messages can be obtained from the GET /cralwers route.

| Name     | Type   | Required | Description                                                                                                             |
|----------|--------|----------|-------------------------------------------------------------------------------------------------------------------------|
| registry | string | required | Performs the analysis of `ALL` Images from the given registry with the option to filter based on a list of repos or tags. |
| k8s      | string | required | Gathers all running container information including namespace annotation information. Setting the Scan field to `true` will also attempt to scan images from their source registry.   |

#### Example Registry Request

*NOTE: Public registry crawling is not supported however external registries can be crawled using the following example.*

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

#### Example K8s Request

*NOTE: KubeConfig field is a takes in a flattened and* **base64** *encoded string. (`kubectl config view --flatten=true | base64 | pbcopy`)*

```
curl -X POST \
  http://192.168.99.100:31700/v1/crawlers/k8s  \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  -d '{
      "K8sCrawler":
      {
        "KubeConfig": "",
        "Context": "minikube",
        "Scan": true,
        "MaxThreads": 10
      }
    }'
```

#### Example Response

```json
{
    "Crawler": {
        "ID": 78,
        "Type": "k8s",
        "Scan": "true",
        "Started": "2018-01-02T14:24:16.908283051-06:00",
        "Finished": "0001-01-01T00:00:00Z"
    }
}
```

### GET /crawlers/`:id`

#### Description

The GET route for the Crawlers resource displays the current state of the crawler. You will need to use ID given from the POST crawler response like in the example above.

The crawler will be in one of 4 states:

| Status       | Description |
|--------------|-------------|
| initializing | An id has been created to track the crawler. |
| started      | Either a k8s or registry scan has begun. |
| scanning     | Scanning within a crawl has commenced. |
| finished     | Completed successfully. A summary of the crawl will be contained within the messages field. |
| error        | An error has occurred within the scan. A detailed error should be contained within the message field. |



#### Example Request

```
curl \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  http://192.168.99.100:31700/v1/crawlers/2
```

#### Example Response

```json
{
    "Crawler": {
        "ID": 59,
        "Type": "k8s",
        "Status": "finished",
        "Messages": {
            "Summary": "** 10 images in https://192.168.99.100:8443 processed in 2.18162059s ** Scan Details: 10 Successful -- 0 Failed -- 0 Skipped",
            "Error": ""
        },
        "Started": "2017-12-22T16:09:53.147508Z",
        "Finished": "2017-12-22T16:09:55.331199Z"
    }
}
```

## K8s-Image-Policy-Webhook

### POST /k8s-image-policy-webhook

#### Description

The POST route for the K8s Image Policy Webhook responds to requests from a Kubernetes cluster that has implemented the [ImagePolicyWebhook Admission Controller](https://kubernetes.io/docs/admin/admission-controllers/#imagepolicywebhook).

The response will contain an allow or deny for images that have vulnerabilities that have violated a given policy.

For instructions on how to setup Minikube to use the webhook can be found [HERE](webhook-example/README.md).

#### Example Request

```
curl -X POST \
  http://192.168.99.100:31700/v1/k8s-image-policy-webhook \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  -d '{
      "kind": "ImageReview",
      "apiVersion": "imagepolicy.k8s.io/v1alpha1",
      "metadata": {
        "creationTimestamp": null
      },
      "spec": {
        "containers": [
          {
            "image": "https://registry-1.docker.io/library/postgres:latest"
          }
        ],
        "annotations": {
          "alpha.image-policy.k8s.io/policy": "default",
          "alpha.image-policy.k8s.io/portauthority-webhook-enable": "true"
        },
        "namespace": "default"
      }
    }'
```

#### Example ALLOW Response

```json
{
  "apiVersion": "imagepolicy.k8s.io/v1alpha1",
  "kind": "ImageReview",
  "status": {
    "allowed": true
  }
}
```

#### Example DENY Response

```json
{
  "apiVersion": "imagepolicy.k8s.io/v1alpha1",
  "kind": "ImageReview",
  "status": {
    "allowed": false,
    "reason": "Scan policy \"default\" detected \"3\" violations for image: https://registry-1.docker.io/library/postgres:latest"
  }
}
```
#### Annotation Parameters

| Name                                                     | Type   | Required | Description                                                       |
|----------------------------------------------------------|--------|----------|-----------------------------|
| alpha.image-policy.k8s.io\policy                         | string   | optional | Allows a custom policy to be applied to the vulnerability review. Otherwise, the policy is set to (default) which will create a violation for ANY detected vulnerability.    |
| alpha.image-policy.k8s.io\portauthority-webhook-enable | string   | optional | Setting "true or false" here will override the default behavior of the server side webhook configuration.    |
