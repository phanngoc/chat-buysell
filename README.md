# Build chat app dùng để đăng tin mua bán.

## Sản phẩm này làm về gì ?

- Người dùng nhắn tin đăng bán + đăng mua, hệ thống tự động dùng NPL để phân loại tin và đưa ra seller, buyer matching tương ứng, User lựa chọn xong sẽ matching tiến hành chat tiếp tục 
để trao đổi + đồng thời hệ thống sẽ tự động gửi tin nhắn SMS cho người mua hoặc người bán (là owner của topic đó), đồng thời gửi notification thông qua FCM với shorcut @{username}

## Sản phẩm cho ai ?
- Người mua và người bán sản phẩm.

## Sản phẩm giải quyết vấn đề gì ?
Khác gì với chợ tốt hay mua bán online truyền thống.
- Không có form đăng tin, chỉ có tin nhắn => ** Nhanh, gọn nhẹ **
- Tự động phân loại tin và matching => ** Công cụ matching bên mua bán với ít thao tác nhất nhất, có thể nhờ AI agent viết truy vấn elasticsearch **
- Không cần vào nhiều forum để tìm tin, hay chọn categories, chỉ cần chat ra lệnh.
- Tự động gửi tin nhắn SMS cho người mua hoặc người bán (là owner của topic đó), gửi notification thông qua FCM ** Thông báo với ít bước nhất, dễ dàng touch đến khách hàng **

## Hướng dẫn chạy thử

1. Cài đặt Go và MongoDB.
2. Đảm bảo MongoDB chạy ở `mongodb://localhost:27017`.
3. Đăng ký Facebook App để lấy `Client ID` và `Client Secret`.
4. Tạo file `.env` hoặc export biến môi trường:

```
export FACEBOOK_CLIENT_ID=your_facebook_client_id
export FACEBOOK_CLIENT_SECRET=your_facebook_client_secret
```

5. Cài đặt package:
```
go mod tidy
```
6. Chạy server:
```
go run main.go
```
7. Truy cập `http://localhost:8080/auth/facebook` để đăng nhập bằng Facebook.

## Cấu trúc MongoDB:
- Database: `chatbuysell`
- Collection: `users`
- Document ví dụ:
```
{
  "uid": "facebook_id",
  "email": "user@email.com",
  "avatar": "https://...",
  "accessToken": "..."
}
```

## API
- `GET /auth/facebook`: Redirect đến Facebook login
- `GET /auth/facebook/callback`: Xử lý callback, trả về thông tin user
