```
service cloud.firestore {
  match /databases/{database}/documents {
  
	match /chatMeta/{userId} {
      allow read, update, delete: if isReqUser(userId);
      allow create: if isSignedIn(); 
      
      // partner can update another user's unread count for initiated chats
      // user can listen on it's own /partners for new chats
      match /partners/{partnerId} {
      	allow read, update, delete: if isReqUser(partnerId) || isReqUser(userId);
        allow create: if isSignedIn();
      }
    }
    
    // both partners in the chat can edit messages
    match /chatMessage/{chatId} {
      match /messages/{messageId} {
   		allow read, update, delete: if inChat(chatId);
      	allow create: if isSignedIn();
      }
    }
    
    // chatId format: [user1]_[user2]
    function inChat(chatId) {
      return isReqUser(chatId.split('_')[0]) 
        || isReqUser(chatId.split('_')[1])
    }
    
    function isReqUser(userId) {
      return request.auth.uid == userId
    }
    
    function isSignedIn() {
	  return request.auth.uid != null;
    }
  }
}
```