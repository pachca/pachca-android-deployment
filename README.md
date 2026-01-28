- gitlab webhook
    - completed Google Play internal release
    - completed Google Play release promotion
    - completed Google Play release update
    - completed all releases
    
- pachca webhook
    - clicked a button
        - open promote Google Play release
        - open update Google Play release
        - open complete all releases
    - submitted form
        - submit promote Google Play release
        - submit update Google Play release
        - submit complete all releases
        

// Promotion can upload release notes as well from app_pachca/play/src/prod/play/release-notes/ru-RU/default.txt
:app_pachca:play:promoteProdArtifact --from-track internal --promote-track alpha --release-status inProgress --user-fraction .25
