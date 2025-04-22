Dưới đây là các **prompt step-by-step** được thiết kế để sử dụng với **Cursor AI Editor** nhằm phát triển một **chat app đăng tin mua bán**, có sử dụng các công nghệ như Golang, MongoDB, Firebase, NLP, Next.js, SMS. Mỗi prompt tập trung vào một phần chức năng cụ thể của hệ thống. Bạn có thể copy từng prompt và chạy trực tiếp trong Cursor để tạo mã nguồn.

---

### **🧩 Step 1: Authentication với Facebook**
**Prompt:**
```
Viết code backend Golang để thực hiện đăng nhập/đăng ký qua Facebook OAuth 2.0, sử dụng mongodb để lưu thông tin người dùng (UID, email, avatar, accessToken). Tạo route `/auth/facebook` để redirect, và `/auth/facebook/callback` để xử lý callback.
```

---

### **💬 Step 2: Cấu trúc MongoDB cho chat và tin đăng**
**Prompt:**
```
Thiết kế schema MongoDB để lưu thông tin:
- User (uid, username, avatar, type: buyer/seller, ...)
- Post (type: 'mua' | 'ban', content, createdAt, userId)
- ChatRoom (roomId, buyerId, sellerId, postId, messages)
- Message (roomId, senderId, content, createdAt)

Xuất code Golang sử dụng `go.mongodb.org/mongo-driver` để tạo các struct và tương tác MongoDB.
```

---

### **🧠 Step 3: Phân loại tin đăng bằng NLP**
**Prompt:**
```
Viết hàm phân loại nội dung tin đăng bằng NLP (vd: "Tôi cần mua iPhone 13 cũ") để xác định loại tin là `mua` hay `bán`. 
Sử dụng thư viện openai để trích xuất thông tin cụ thể gồm:
   - category: loại sản phẩm (ví dụ: điện thoại, laptop, xe máy, nhà,...)
   - location: địa điểm trong nội dung (ví dụ: Hà Nội, TP.HCM,...)
   - price: số tiền (nếu có, dạng số nguyên không có đơn vị)
   - condition: tình trạng sản phẩm nếu có (ví dụ: mới, cũ, like new,...)
   - keywords: danh sách 3-5 từ khóa mô tả sản phẩm
```

---

### **🔍 Step 4: Matching người mua và người bán**
**Prompt:**
```
Viết hàm matching giữa người mua và người bán dựa trên nội dung tin đăng. Dùng MongoDB để tìm các tin đăng ngược loại (buyer tìm seller và ngược lại) có nội dung tương đồng. Tạo API Golang để trả về danh sách user matching.
```

---

### **💡 Step 5: Tạo luồng chat sau khi matching**
**Prompt:**
```
Tạo API Golang `/chat/start` để tạo ChatRoom giữa người mua và người bán sau khi user chọn đối tượng matching. Nếu ChatRoom đã tồn tại thì return ID, nếu chưa có thì tạo mới. Tự động insert bản tin đầu tiên (tin đăng) làm message đầu tiên.
```

---

### **✉️ Step 6: Gửi SMS khi có matching**
**Prompt:**
```
Tạo hàm trong Golang để gửi SMS đến số điện thoại người bán hoặc người mua khi có ChatRoom mới được tạo. Dùng Twilio hoặc dịch vụ gửi SMS tương thích. Nội dung SMS nên bao gồm: "{buyer/seller} đã quan tâm đến tin đăng của bạn: {content}".
```

---

### **📲 Step 7: Gửi FCM push notification**
**Prompt:**
```
Viết hàm Golang gửi FCM push notification khi có người nhắn tin trong chat. Thông báo bao gồm avatar, nội dung tin nhắn cuối cùng, shortcut dạng "@{username}". Dùng Firebase Admin SDK để gửi notification từ server.
```

---

### **📱 Step 8: Tạo UI frontend bằng Next.js**
**Prompt:**
```
Tạo giao diện Next.js để hiển thị danh sách tin đăng (mua/bán) và cho phép người dùng nhắn tin với người matching. Giao diện bao gồm:
- Trang đăng nhập Facebook
- Trang đăng tin (mua/bán)
- Trang danh sách người dùng matching
- Chatroom UI

Sử dụng TailwindCSS và Firebase Authentication client để login.
```

---

### **🧠 BONUS: Tích hợp AI phân loại và gợi ý mô tả tin**
**Prompt:**
```
Tạo endpoint Golang nhận nội dung tin nhắn và sử dụng OpenAI API để phân loại + gợi ý mô tả chi tiết hơn cho tin đăng (ví dụ: "Bạn muốn mua iPhone cũ? Hãy ghi rõ dung lượng, màu sắc, tình trạng..."). Output là JSON gồm loại tin (`mua` hoặc `bán`) và nội dung gợi ý cải tiến.
```

---

Bạn muốn mình đóng gói những prompt này thành một **tài liệu kỹ thuật** hay là tạo sẵn **một repo mẫu** để khởi động nhanh dự án này?