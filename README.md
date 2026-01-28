# Pachca Android deployment 

## Actors

- **Gitlab** runs actual CI/CD, builds bundles and APKs, uploads them to corresponding app stores.
- **Pachca** serves as a frontend for interaction with **this service** via messages with actions in **internal chat**.
- **This servce** acts as a mediator between **Gitlab** and **Pachca**, receiving hooks from one of them to run an action on another.

## CI/CD Workflow

### Request is merged into release branch in **Gitlab**

- **Gitlab** builds a release bundle and uploads it to Google Play internal track.
- **This service** receives a hook from **Gitlab** with the result of upload, job id, versionCode and versionName of the build.
- **This service** sends a message to **internal chat** a button "Promote release" in Pachca and pins it.


### "Promote build" message button is clicked in **internal chat** 

- **This service** receives a hook from **Pachca** with the info from the button.
- **This service** opens a form in **Pachca** with two fields: rollout percentage and release notes.
- **This service** receives a hook from **Pachca** with the filled out form and launches a **Gitlab** job that uploads release notes, promotes release to production and sets rollout percentage.


### Build is promoted to production in **Gitlab**

- **This service** receives a hook from **Gitlab** with the result of the promotion.
- **This service** updates the message in **internal chat** with text stating that the build is in production track, and two buttons: "Update rollout" (if not 100% yet) and "Release to all stores"


### "Update rollout" message button is clicked in **internal chat**

- **This service** receives a hook from **Pachca** with the info from the button.
- **This service** opens a form in **Pachca** with one field: rollout percentage.
- **This service** receives a hook from **Pachca** with the filled out form and launches a **Gitlab** job that sets rollout percentage.


### Rollout percentage is updated in **Gitlab**

- **This service** receives a hook from **Gitlab** with the result of the rollout update.
- **This service** updates the message in **internal chat** with text with new rollout percentage, and two buttons: "Update rollout" (if not 100% yet) and "Release to all stores"


### "Release to all stores" message button is clicked in **internal chat**

- **This service** receives a hook from **Pachca** with the info from the button.
- **This service** opens a form in **Pachca** with one fields: release notes.
- **This service** receives a hook from **Pachca** with the filled out form and launches a **Gitlab** job that builds bundles for other stores and releases them.


### Releases to other stores are completed in **Gitlab**

- **This service** receives a hook from **Gitlab** with the result of the uploads.
- **This service** updates the message in **internal chat** with text that all is complete and no buttons, then unpins the message.

---

Promotion can upload release notes as well from app_pachca/play/src/prod/play/release-notes/ru-RU/default.txt
:app_pachca:play:promoteProdArtifact --from-track internal --promote-track alpha --release-status inProgress --user-fraction .25
