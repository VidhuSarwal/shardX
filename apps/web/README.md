# Welcome to your Lovable project

## Project info

**URL**: https://lovable.dev/projects/46337819-cb24-409b-a258-ba9438035066

## How can I edit this code?

There are several ways of editing your application.

**Use Lovable**

Simply visit the [Lovable Project](https://lovable.dev/projects/46337819-cb24-409b-a258-ba9438035066) and start prompting.

Changes made via Lovable will be committed automatically to this repo.

**Use your preferred IDE**

If you want to work locally using your own IDE, you can clone this repo and push changes. Pushed changes will also be reflected in Lovable.

The only requirement is having Node.js & npm installed - [install with nvm](https://github.com/nvm-sh/nvm#installing-and-updating)

Follow these steps:

```sh
# Step 1: Clone the repository using the project's Git URL.
git clone <YOUR_GIT_URL>

# Step 2: Navigate to the project directory.
cd <YOUR_PROJECT_NAME>

# Step 3: Install the necessary dependencies.
npm i

# Step 4: Start the development server with auto-reloading and an instant preview.
npm run dev
```

**Edit a file directly in GitHub**

- Navigate to the desired file(s).
- Click the "Edit" button (pencil icon) at the top right of the file view.
- Make your changes and commit the changes.

**Use GitHub Codespaces**

- Navigate to the main page of your repository.
- Click on the "Code" button (green button) near the top right.
- Select the "Codespaces" tab.
- Click on "New codespace" to launch a new Codespace environment.
- Edit files directly within the Codespace and commit and push your changes once you're done.

## What technologies are used for this project?

This project is built with:

- Vite
- TypeScript
- React
- shadcn-ui
- Tailwind CSS

## Downloading files with your key file

1. Open the **Download** page in the app navigation (you must be signed in).
2. Select the `.2xpfm.key` file you saved after a successful upload.
3. Click **Start download** to initiate a reconstruction session. The UI automatically polls the backend every 3 seconds and shows progress, ETA, and checksum metadata.
4. When the status reaches _Complete_, the browser starts downloading the reconstructed file and, when possible, verifies the SHA-256 checksum that the backend provides.
5. You can cancel, retry, or reset the flow at any point. Session IDs expire automatically; if that happens, simply upload the key file again to begin a fresh download session.

## How can I deploy this project?

Simply open [Lovable](https://lovable.dev/projects/46337819-cb24-409b-a258-ba9438035066) and click on Share -> Publish.

## Can I connect a custom domain to my Lovable project?

Yes, you can!

To connect a domain, navigate to Project > Settings > Domains and click Connect Domain.

Read more here: [Setting up a custom domain](https://docs.lovable.dev/features/custom-domain#custom-domain)
