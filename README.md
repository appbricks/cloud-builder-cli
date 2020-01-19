# Cloud Builder Automation Command Line Interface

[![Build Status](https://travis-ci.org/appbricks/cloud-builder-cli.svg?branch=master)](https://travis-ci.org/appbricks/cloud-builder-cli)

## Overview

[Cloud automation](https://github.com/appbricks/cloud-builder) recipes can be launched using this CLI which launches these recipes from within you local shell. The CLI allows you to securely save your cloud credentials and configurations locally or remotely. It will execute all recipes locally and save state remotely based on the recipe's specification unless it is configured explicitly to be saved locally.

## Command Tree

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
   ├─ cloud - The cloud-builder CLI includes a set of recipes that can be launched
   │    │     in the public cloud. The commands below allow you to retrieve
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
   ├─ recipe - The cloud-build CLI includes a set of recipes which contain
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
        │         been configured for.
        │
        ├─ show - Show the deployment configuration values for the target. If the
        │         target has not been created and configured then this sub-command will
        │         return an error. Run 'cb target list' to view the list of configured
        │         targets.
        │
        ├─ create - Creates and configures a quick launch target for a given recipe and 
        │           cloud.
        │
        ├─ configure - Configures an existing quick launch target. Once configure target
        │              will need to be re-launched for any configuration changes to take
        │              effect.
        │
        ├─ launch - Deploys a quick launch target or re-applies any configuration updates.
        │
        ├─ ssh - SSH to the target environment. This is for advance users as well as 
        │        for troubleshooting any configuration errors at the target. If the 
        │        target consists of more than one instance this will create a secure
        │        shell to primary instance identified by the cloud recipe of the 
        │        target.
        │
        ├─ suspend - Suspends all instance resources deployed to a target.
        │
        ├─ resume - Resumes hibernated resources at a deployed target.
        │
        ├─ **migrate - Migrates services at a given target to different target.
        │
        ├─ **share - Shares access to a target with another registered user.
        │
        └─ delete - Deletes a deployed target.

```

> ** Sub-command still to be implemented
