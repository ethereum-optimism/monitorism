# Jupyter Playbook for Incident Response

This repository contains Jupyter notebooks designed to help manage and streamline incident response processes. Jupyter notebooks offer an interactive, visual environment that can assist in documenting and automating various steps during incidents, making them an ideal tool for incident response teams.

## Why Use Jupyter Notebooks for Incident Response?

Jupyter notebooks allow for a flexible and dynamic response to incidents by combining live code, notes, and visualizations in one place. They are particularly helpful in:

- **Documenting steps**: Keep a real-time log of actions taken during incident resolution.
- **Automation**: Execute code directly within the notebook to gather information, analyze logs, or perform specific tasks.
- **Collaboration**: Share the notebook across teams or incident responders to maintain consistent actions and responses.

## How to Use

To run these notebooks locally:

1. Clone the repository.
2. Run the `make start` command, which will launch the notebooks in your local environment, allowing you to start your incident response process.

Alternative way to use it is directly in visual studio code. Video below show how to use it. You only need to click on the playbook at the start of the playbook (after eventually set the variable needed) and press CMD+Enter or CTRL+Enter and execute one block at a time.

https://github.com/user-attachments/assets/5800dd75-58f2-4114-b8d8-dc187009376e


## Setting Variables

Before starting, you will need to configure some local variables for the notebooks to function correctly. These variables can be set in your local environment or directly within the text of the notebook. To avoid setting environment variables repeatedly for multiple runbooks, you can store them in a `.env` file located in the same folder as the notebooks.

There is an example file available for your convenience (`env.example`) that you can use to create your `.env` file and adjust it as needed. This will help streamline the process of setting up your environment variables for different playbooks.

## Improving Productivity

As you develop new actions or workflows during incidents, you can save them within the notebooks and push the updates to Git. This allows the incident response process to evolve and improve continuously, helping to enhance productivity and ensure all team members have access to the latest procedures.

## ⚠️ Warning

When committing runbooks back to the repository, **make sure not to commit any runs or logs containing sensitive data**. Review the content carefully to ensure no private information is included before pushing to Git.
