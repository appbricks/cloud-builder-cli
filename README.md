# Cloud Builder Automation Command Line Interface

[![Build Status](https://github.com/appbricks/cloud-builder-cli/actions/workflows/build-dev-release.yml/badge.svg)](https://github.com/appbricks/cloud-builder-cli/actions/workflows/build-dev-release.yml)
[![Build Status](https://github.com/appbricks/cloud-builder-cli/actions/workflows/build-prod-release.yml/badge.svg)](https://github.com/appbricks/cloud-builder-cli/actions/workflows/build-prod-release.yml)

## Overview

This CLI allows you to launch [Cloud automation](https://github.com/appbricks/cloud-builder) recipes from within your local shell. It saves your cloud credentials and configurations locally and will execute recipes saving their deployment state remotely based on the recipe's specification. Cloud credentials are encrypted and saved locally using a key you provide at initialization. Once recipes have been deployed the CLI can be used to managed the life-cycle of the deployed services.

Future iterations of the CLI will included a registration requirement, which will Single Sign-On (SSO) with the [AppBricks.io](https://appbricks.io) domain. This will require you maintain a subscription account in order to access advance features such as a cost optimization engine, the ability to view telemetry on deployed services, and be alerted on events that might impact the cost, performance and security of services in your sandbox. The subscription will provide a free tier which will allow you to launch and run basic services such as the VPN in your sandbox.

## Installation

The CLI is available for 64 bit versions of [Apple macOS](https://en.wikipedia.org/wiki/MacOS), [Microsoft Windows](https://en.wikipedia.org/wiki/Microsoft_Windows) and [Linux](https://en.wikipedia.org/wiki/Linux). You can download the [zip](https://en.wikipedia.org/wiki/Zip_(file_format)) archive of the appropriate binary for you operating system from the [releases](https://github.com/appbricks/cloud-builder-cli/releases) page of this repository.

Once downloaded unzip the downloaded archive and copy the binary file named `cb` to a system path location. Copying the file to the following system paths will make the binary available globally within any shell in your system. You will need to provide root or admin privileges to copy the CLI binary to these paths.

|Operating System     |Path              |
|---------------------|------------------|
| Apple MacOS         | `C:\Windows`     |
| Microsoft Windows   | `/usr/local/bin` |
| Linux (i.e. Ubuntu) | `/usr/local/bin` |

Alternatively copy the CLI binary to a user directory and add it to the system path environment variable.

## Creating your Public Cloud Accounts

In order to deploy your sandbox in one of the public cloud regions you need to have an active account in all or one of the following cloud providers.

* [Amazon Web Services (AWS)](doc/aws.md)
* [Microsft Azure](doc/azure.md)
* [Google Cloud Platform (GCP)](doc/google.md)

## Usage

Once the Cloud Builder CLI has been downloaded and extracted into your system path you can invoke it from shell or command window from you home folder. You can retrieve help for any command or sub-command by adding the global option `--help` to any command. It is recommended that you run `cb init` before configuring the CLI as that will configure encryption of your cloud credentials and target state.

> The current release of the `cb` CLI does not ask you to register or associate with an [AppBricks.IO](https://appbricks.io) account when you run `init`. This may change in future releases.

Once you have accepted the license and initialized the context you need configure your public cloud account where you plan to build your sandbox. You can choose to configure all three available cloud providers in order to maximize the regions where you can build you sandboxes. Whenever you deploy resources keep in mind that live resources, such as compute, will run up costs in your cloud account. It is always a good idea to enable the recipe's auto shutdown attribute or explicitly shutdown the target via the CLI.

### Configuring Cloud Accounts

Before you can begin launching your cloud targets you need to make sure the CLI is configured with your public cloud credentials. Each public cloud has its own specific credential attributes that need to be provided.

* To list the available public clouds that can be configured run the following.

  ```
  ❯ cb cloud list

  This Cloud Builder cookbook supports launching recipes in the public clouds listed below.

  +--------+------------------------------------------+------------+
  | Name   | Description                              | Configured |
  +--------+------------------------------------------+------------+
  | aws    | Amazon Web Services Cloud Platform       | no         |
  | azure  | Microsoft Azure Cloud Computing Platform | no         |
  | google | Google Cloud Platform                    | no         |
  +--------+------------------------------------------+------------+
  ```

  If you want to retrieve the list of regions for each public cloud then run.

  ```
  cb cloud list -r
  ```

* You can see the list of required inputs to configure a public cloud with the following command.

  ```
  ❯ cb cloud show aws

  Cloud Provider Configuration
  ============================

  Amazon Web Services Cloud Platform

  CONFIGURATION DATA INPUT REFERENCE

  * Access Key - The AWS account's access key id. It will be sourced from the
                 environment variable AWS_ACCESS_KEY_ID if not provided.
  * Secret Key - The AWS account's secret key. It will be sourced from the
                 environment variable AWS_SECRET_ACCESS_KEY if not provided.
  * Token      - AWS multi-factor authentication token. It will be sourced from
                 the environment variable AWS_SESSION_TOKEN if not provided.
  * Region     - The AWS region to create resources in. It will be sourced from
                 the environment variable AWS_DEFAULT_REGION if not provided.
                 (Default value = 'us-east-1')
  ```

  > For cloud configurations most inputs can source their values from environment variables if provided. Some inputs may be a path to a file, like the service key file for Google Cloud Platform (GCP).

* To configure your cloud account run the following.

  ```
  ❯ cb cloud configure aws

  Cloud Provider Configuration
  ============================

  Amazon Web Services Cloud Platform

  CONFIGURATION DATA INPUT
  ================================================================================

  Access Key - The AWS account's access key id. It will be sourced from the
              environment variable AWS_ACCESS_KEY_ID if not provided.
  --------------------------------------------------------------------------------
  : ****

  Secret Key - The AWS account's secret key. It will be sourced from the
              environment variable AWS_SECRET_ACCESS_KEY if not provided.
  --------------------------------------------------------------------------------
  : ****

  Token - AWS multi-factor authentication token. It will be sourced from the
          environment variable AWS_SESSION_TOKEN if not provided.
  --------------------------------------------------------------------------------
  :

  Region - The AWS region to create resources in. It will be sourced from the
          environment variable AWS_DEFAULT_REGION if not provided.
  --------------------------------------------------------------------------------
  : us-east-1


  Configuration input saved
  ```

  > When you are prompted for a particular input the `TAB` key will let you cycle through the list of possible inputs sourced from the environment or an input list of valid values (i.e. cloud regions).

### Configuring Recipes

Recipes like the default `sandbox` recipe come with pre-configured defaults. For example, by default it will deploy a sandbox virtual private cloud (VPC) with an OpenVPN service which will route traffic between the DMZ, Admin network and Internet. Once logged in to the VPN it will tunnel all traffic to the internet from your device via the VPC's DMZ.

> The sandbox is a useful alternative to having your own dedicated free VPN server rather than subscribing to a hosted VPN provider like [ExpressVPN](https://www.expressvpn.com/). Having you own VPN server will ensure greater levels of privacy and anonymity as well as performance issues on overloaded servers that are being shared by many users concurrently. The 'ovpn-x' is a special deployment of an OpenVPN bastion instance that runs a proxy that will mask OpenVPN traffic to bypass DPI routers that block VPN traffic at your internet service provider.

* To list the available recipes that can be deployed to your cloud account run the following command.

  ```
  ❯ cb recipe list

  This Cloud Builder cookbook supports launching the following recipes.

  +---------+-----------------------------------------------+------------------+
  | Name    | Description                                   | Supported Clouds |
  +---------+-----------------------------------------------+------------------+
  | sandbox | My Cloud Space virtual private cloud sandbox. | aws              |
  |         |                                               | azure            |
  |         |                                               | google           |
  +---------+-----------------------------------------------+------------------+
  ```

  > By default only the sandbox recipe is available. Additional recipes will need to be imported from the marketplace or a download archive of the recipe.

* Recipes are preconfigured with default values. These values can be viewed or changed with the following commands.

  To view the recipe inputs and their default values for a particular cloud run.

  ```
  cb recipe show sandbox aws
  ```

  To change these values run.

  ```
  cb recipe configure sandbox aws
  ```

  > The default recipe creates the same default VPN user and password. It is important that you reconfigure all the recipes with your unique VPN users and passwords before launching sandbox VPCs.

### Configuring and Launching Targets

Targets are recipes that have been configured to be launched to a particular cloud and region.

* To list the targets run the following.

  ```
  ❯ cb target list

  The following recipe targets have been configured.

  +---------+--------+--------+-----------------+---------+----------------+---+
  | Recipe  | Cloud  | Region | Deployment Name | Version | Status         | # |
  +---------+--------+--------+-----------------+---------+----------------+---+
  | sandbox | aws    |        |                 |         | not configured |   |
  +---------+--------+--------+-----------------+---------+----------------+---+
  |         | azure  |        |                 |         | not configured |   |
  +---------+--------+--------+-----------------+---------+----------------+---+
  |         | google |        |                 |         | not configured |   |
  +---------+--------+--------+-----------------+---------+----------------+---+
  ```

  Initially no targets will be configured.

* Configure your first target with the following command.

  ```
  cb target create sandbox aws
  ```

  This command will cycle through a list of inputs similar to the cloud or recipe configurations. You can accept all the cloud and region specific defaults unless you want to customize for example the bastion instance type.

* Once a target has been configured the target list will show the cloud and region a target has been configured for.

  ```
  ❯ cb target list

  The following recipe targets have been configured.

  +---------+--------+-----------+-----------------+---------+----------------+---+
  | Recipe  | Cloud  | Region    | Deployment Name | Version | Status         | # |
  +---------+--------+-----------+-----------------+---------+----------------+---+
  | sandbox | aws    | us-east-1 | MyVPN           |         | not deployed   | 1 |
  +---------+--------+-----------+-----------------+---------+----------------+---+
  |         | azure  |           |                 |         | not configured |   |
  +---------+--------+-----------+-----------------+---------+----------------+---+
  |         | google |           |                 |         | not configured |   |
  +---------+--------+-----------+-----------------+---------+----------------+---+
  ```

  Also displayed with be the launch or deployment status of the target.

### Managing Deployed Targets

The `cb target ...` sub-commands are used to manage the lifecycle of configured and/or deployed targets.

* Running the following command will bring up interactive prompts that will allow you to run target sub-commands without having to provide the entire CLI command.

  ```
  ❯ cb target list -e

  The following recipe targets have been configured.

  +---------+--------+-----------+-----------------+---------+----------------+---+
  | Recipe  | Cloud  | Region    | Deployment Name | Version | Status         | # |
  +---------+--------+-----------+-----------------+---------+----------------+---+
  | sandbox | aws    | us-east-1 | MyVPN           |         | not deployed   | 1 |
  +---------+--------+-----------+-----------------+---------+----------------+---+
  |         | azure  |           |                 |         | not configured |   |
  +---------+--------+-----------+-----------------+---------+----------------+---+
  |         | google |           |                 |         | not configured |   |
  +---------+--------+-----------+-----------------+---------+----------------+---+

  Enter # of node to execute sub-command on or (q)uit: 1

  Select sub-command to execute on target.

    Recipe: sandbox
    Cloud:  aws
    Region: us-east-1
    Name:   MyVPN

  1 - Show
  2 - Configure
  3 - Launch
  4 - Delete
  5 - Suspend
  6 - Resume

  Enter # of sub-command or (q)uit:
  ```

* Run target show to get current deployed state of a target as well as instructions on retrieving the VPN configurations.

  ```
  ❯ cb target show sandbox aws us-east-1 MyVPN

  Deployment: MYVPN
  ==================

  Status: running

  This My Cloud Space sandbox has been deployed to the following public
  cloud environment. Along with a sandboxed virtual cloud network it
  includes a VPN service which allows you to access the internet as
  well as your personal cloud space services securely while maintaining
  your privacy.

  Provider: Amazon Web Services
  Region: us-east-1
  VPN Type: OpenVPN
  Version: 0.0.1

  Instance: bastion
  ------------------

  State: running

  The Bastion instance runs the VPN service that can be used to
  securely and anonymously access your cloud space resources and the
  internet. You can download the VPN configuration along with the VPN
  client software from the password protected links below. The same
  user and password used to access the link should be used as the login
  credentials for the VPN.

  * URL: https://23.22.130.58/~user1
    User: user1
    Password: ****
  ```

  You can run the following command to display a comprehensive list of the configuration attributes of the sandbox recipe used in the deployment along with the status show above.

  ```
  cb target show sandbox aws us-east-1 MyVPN -a
  ```

* Run the following command to destroy all resources created by a launched target. You can provide the '-k' option if you want to ensure the target configuration is preserved.

  ```
  cb target delete sandbox aws us-east-1 MyVPN
  ```

## Command Reference Tree

The following command reference outlines all the available CLI commands and indicates which ones are available for space admins vs. guests.

```
  cb
   ├─ init - This will register or associate a cloud builder user with all CLI
   │         sessions. You need to register if you would like to share access to
   │         targets or would like to synchronize access to configurations across
   │         all your devices. It will also create client specific keys for
   │         encryption of cloud configurations. All credentials including
   │         configuration information are encrypted using public-private key
   │         encryption. When you initialize the CLI for first time the keys will
   │         be created and your private key will be saved to you system's key
   │         store. You will need to add this key to each of your devices from
   │         which you want to interact with or control your launch targets.
   │
   ├─ logout - Signs out the current user in context.
   │
   ├─ cloud - (admin) The cloud-builder CLI includes a set of recipes that can be
   │    │     launched in the public cloud. The commands below allow you to retrieve
   │    │     information regarding these cloud environments and configure them as
   │    │     launch targets for the recipes.
   │    │
   │    ├─ list - Show a list of public clouds and region information where recipes can
   │    │         be launched. In order to be able to target a recipe to one of these
   │    │         clouds you need to have a valid account with the correct permissions.
   │    │
   │    ├─ show - Show detailed information regarding the given cloud. This command
   │    │         will also show help for the configuration data required for the
   │    │         given cloud.
   │    │
   │    └─ configure - Recipe resources are created in the public cloud using your cloud
   │                   credentials. This requires that you have a valid cloud account in one
   │                   or more of the clouds the recipe can be launched in. This command can
   │                   be used to configure your cloud credentials for the cloud environments
   │                   you wish to target.
   │
   ├─ recipe - (admin) The cloud-build CLI includes a set of recipes which contain
   │    │      instructions on launching a services in the cloud. The sub-commands
   │    │      below allow interaction with recipe templates to create customized
   │    │      targets which can be launched on demand.
   │    │
   │    ├─ list - Lists the recipes bundled with the CLI that can be launched in any
   │    │         one of the supported public clouds.
   │    │
   │    ├─ show - Shows information regarding the given cloud recipe. This command will
   │    │         also show help for all the recipe inputs including defaults that can
   │    │         be provided to customzie the deployment of the recipe.
   │    │
   │    ├─ **import - Import cloud recipes from Github or a downloaded zip archive. This
   │    │             command allows you extend your personal cloud sandbox with additional
   │    │             application and services securely.
   │    │
   │    └─ configure - Recipes are parameterized to accomodate different configurations in
   │                   the cloud. This command can be used to create a standard template
   │                   which can be further customized when configuring a target.
   │
   └─ target - A target is an instance of a recipe that can be launched with a single
        │      click to a cloud region. When a recipe is configured for a particular
        │      cloud it will  enumerate all the regions of that cloud as quick lauch
        │      targets. The sub-commands below allow you to launch and view the status
        │      of targets.
        │
        ├─ list - List all available quick launch targets which are configured recipe
        │         instances. Targets will be enumerated only for clouds a recipe has
        │         been configured for and the logged in user has access to.
        │
        ├─ show - Show the deployment configuration values for the target. If the
        │         target has not been created and configured then this sub-command will
        │         return an error. Run 'cb target list' to view the list of configured
        │         targets.
        │
        ├─ create - (admin) Creates and configures a quick launch target for a given recipe
        │           and cloud.
        │
        ├─ configure - (admin) Configures an existing quick launch target. Once configure
        │              target will need to be re-launched for any configuration changes to
        │              take effect.
        │
        ├─ launch - (admin) Deploys a quick launch target or re-applies any configuration
        │           updates.
        │
        ├─ delete - (admin) Deletes a deployed target.
        │
        ├─ suspend - (admin) Suspends all instance resources deployed to a target.
        │
        ├─ resume - (admin) Resumes hibernated resources at a deployed target.
        │
        ├─ connect - Securely connects to a deployed target space.
        │
        ├─ ssh - (admin) SSH to the target environment. This is for advance users as
        │        well as for troubleshooting any configuration errors at the target.
        │        If the target consists of more than one instance this will create a
        │        secure shell to primary instance identified by the cloud recipe of
        │        the target.
        │
        ├─ **migrate - (admin) Migrates services at a given target to different target.
        │
        └─ **share - (admin) Shares access to a target with another registered user.

```

> ** Sub-command still to be implemented

## Generating User Private Keys

When a client device is setup with a device owner the CLI will check if the user is
associated with a key. If the user has not been setup with a key then the CLI will
request a new key be imported or generated. If you wish to create the key manually
you can do it via the following `openssl` commands.

```
openssl genrsa -passout pass:x -out my-private-key-temp.pem -aes256 4096
openssl pkcs8 -topk8 -v2 aes256 -passin pass:x -passout pass:MYSECRET -in my-private-key-temp.pem -out my-private-key.pem
rm my-private-key-temp.pem
```

The second command is required to convert the key to a format the CLI can read. The
key file created would be `my-private-key.pem` from the above example. Once created
you should save the generated key file in a secure location preferrably offline
on a USB stick which can be locked away. You will need this key in the future to 
unlock configurations and claim additional devices.
