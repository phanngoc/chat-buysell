import { useState, useEffect, useRef } from 'react';
import Head from 'next/head';
import { useRouter } from 'next/router';
import axios from 'axios';
import { useAuth } from '../components/AuthContext';
import SearchResults from '../components/SearchResults';

export default function Chat() {
  const { user, loading, logout } = useAuth();
  const router = useRouter();
  const [message, setMessage] = useState('');
  const [messages, setMessages] = useState([]);
  const [chatRooms, setChatRooms] = useState([]);
  const [activeRoom, setActiveRoom] = useState(null);
  const [isLoading, setIsLoading] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [searchResults, setSearchResults] = useState(null);
  const [showSearchResults, setShowSearchResults] = useState(false);
  const messagesEndRef = useRef(null);

  // If not logged in, redirect to home
  useEffect(() => {
    if (!loading && !user) {
      router.push('/');
    } else if (user) {
      // Load chat rooms for the user
      fetchChatRooms();
    }
  }, [loading, user, router]);

  // Scroll to bottom when messages change
  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  // Fetch chat rooms for the user
  const fetchChatRooms = async () => {
    try {
      setIsLoading(true);
      const response = await axios.get(`/api/chat/rooms?userId=${user.id}`);
      setChatRooms(response.data.rooms || []);
      setIsLoading(false);
    } catch (error) {
      console.error('Failed to fetch chat rooms:', error);
      setIsLoading(false);
    }
  };

  // Load messages for a specific room
  const loadChatRoom = async (roomId) => {
    try {
      setIsLoading(true);
      const response = await axios.get(`/api/chat/room/${roomId}`);
      setActiveRoom(response.data.chatRoom);
      setMessages(response.data.messages || []);
      setIsLoading(false);
    } catch (error) {
      console.error('Failed to load chat room:', error);
      setIsLoading(false);
    }
  };

  // Send a message
  const sendMessage = async (e) => {
    e.preventDefault();
    if (!message.trim() || !activeRoom) return;

    try {
      const response = await axios.post('/api/chat/message', {
        roomId: activeRoom.id,
        senderId: user.id,
        content: message
      });

      // Add the new message to the list
      const newMessage = {
        id: response.data.messageId,
        content: message,
        senderId: user.id,
        createdAt: new Date().toISOString()
      };

      setMessages([...messages, newMessage]);
      setMessage('');
    } catch (error) {
      console.error('Failed to send message:', error);
    }
  };

  // Create a new post (buy or sell)
  const createPost = async (type) => {
    try {
      const content = prompt(`Enter your ${type === 'mua' ? 'buying' : 'selling'} post details:`);
      if (!content) return;

      const response = await axios.post('/api/post/create', {
        userId: user.id,
        content,
        type
      });

      alert('Post created successfully!');
      
      // Show matching results
      if (response.data.post) {
        findMatches(response.data.post.content);
      }
    } catch (error) {
      console.error('Failed to create post:', error);
      alert('Failed to create post. Please try again.');
    }
  };

  // Find matching posts based on content
  const findMatches = async (content) => {
    try {
      setIsLoading(true);
      const response = await axios.post('/api/matching/find', {
        content,
        page: 1,
        pageSize: 10
      });

      if (response.data && response.data.matches) {
        setSearchResults(response.data.matches);
        setShowSearchResults(true);
      }
    } catch (error) {
      console.error('Error finding matches:', error);
      alert('Failed to find matching posts.');
    } finally {
      setIsLoading(false);
    }
  };

  // Handle searching for posts
  const handleSearch = async (e) => {
    e.preventDefault();
    if (!searchText.trim()) return;
    
    await findMatches(searchText);
    setSearchText('');
  };

  // Handle a chat room being created from search results
  const handleChatRoomCreated = (newRoom) => {
    setChatRooms([...chatRooms, newRoom]);
    setActiveRoom(newRoom);
    loadChatRoom(newRoom.id);
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <Head>
        <title>Chat | Chat Buy Sell App</title>
        <meta name="description" content="Chat with buyers and sellers" />
      </Head>

      <div className="flex h-screen">
        {/* Sidebar */}
        <div className="w-1/4 bg-white border-r border-gray-200 flex flex-col">
          {/* User info */}
          <div className="p-4 border-b border-gray-200 flex items-center">
            {user?.avatar && (
              <img 
                src={user.avatar} 
                alt={user.username || 'User'} 
                className="w-10 h-10 rounded-full mr-3"
              />
            )}
            <div className="flex-1">
              <h3 className="font-medium">{user?.username || 'User'}</h3>
              <p className="text-sm text-gray-500">{user?.email}</p>
            </div>
            <button 
              onClick={logout} 
              className="text-sm text-red-500 hover:text-red-700"
            >
              Logout
            </button>
          </div>

          {/* Search and post creation */}
          <div className="p-4 border-b border-gray-200">
            <form onSubmit={handleSearch} className="mb-3">
              <div className="flex">
                <input
                  type="text"
                  value={searchText}
                  onChange={(e) => setSearchText(e.target.value)}
                  placeholder="Search for products..."
                  className="input"
                />
                <button 
                  type="submit"
                  className="ml-2 px-3 bg-primary hover:bg-indigo-700 text-white rounded-md"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                  </svg>
                </button>
              </div>
            </form>

            <div className="flex space-x-2">
              <button 
                onClick={() => createPost('mua')} 
                className="flex-1 py-2 px-3 bg-blue-500 text-white rounded-md hover:bg-blue-600"
              >
                Buy Post
              </button>
              <button 
                onClick={() => createPost('ban')} 
                className="flex-1 py-2 px-3 bg-green-500 text-white rounded-md hover:bg-green-600"
              >
                Sell Post
              </button>
            </div>
          </div>

          {/* Chat rooms list */}
          <div className="flex-1 overflow-y-auto">
            <h3 className="px-4 py-2 text-sm font-semibold text-gray-500">CHATS</h3>
            {isLoading && chatRooms.length === 0 ? (
              <div className="flex justify-center p-4">
                <div className="animate-spin rounded-full h-6 w-6 border-t-2 border-b-2 border-primary"></div>
              </div>
            ) : chatRooms.length === 0 ? (
              <p className="px-4 py-2 text-sm text-gray-500">No chat rooms yet</p>
            ) : (
              chatRooms.map(room => (
                <div 
                  key={room.id} 
                  onClick={() => loadChatRoom(room.id)}
                  className={`px-4 py-3 flex items-center cursor-pointer hover:bg-gray-50 ${activeRoom?.id === room.id ? 'bg-gray-100' : ''}`}
                >
                  <div className="w-10 h-10 rounded-full bg-gray-300 flex items-center justify-center text-gray-600 mr-3">
                    {room.type === 'buyer' ? 'B' : 'S'}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex justify-between">
                      <h3 className="font-medium truncate">{room.title || 'Chat Room'}</h3>
                      <span className="text-xs text-gray-500">
                        {new Date(room.updatedAt).toLocaleDateString()}
                      </span>
                    </div>
                    <p className="text-sm text-gray-500 truncate">
                      {room.lastMessage || 'No messages yet'}
                    </p>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Chat area */}
        <div className="flex-1 flex flex-col relative">
          {/* Search results modal */}
          {showSearchResults && (
            <div className="absolute inset-0 bg-black bg-opacity-50 flex items-center justify-center z-10">
              <div className="w-3/4 max-w-2xl">
                <SearchResults 
                  results={searchResults}
                  onClose={() => setShowSearchResults(false)}
                  onChatRoomCreated={handleChatRoomCreated}
                />
              </div>
            </div>
          )}

          {activeRoom ? (
            <>
              {/* Chat header */}
              <div className="p-4 border-b border-gray-200 flex items-center">
                <h2 className="font-medium flex-1">{activeRoom.title || 'Chat'}</h2>
                {activeRoom.post && (
                  <span className="px-2 py-1 bg-gray-100 text-xs rounded-full">
                    {activeRoom.post.type === 'mua' ? 'Buying' : 'Selling'}
                  </span>
                )}
              </div>

              {/* Messages area */}
              <div className="flex-1 overflow-y-auto p-4 flex flex-col">
                {messages.length === 0 ? (
                  <div className="flex-1 flex items-center justify-center">
                    <p className="text-gray-500">No messages yet. Start the conversation!</p>
                  </div>
                ) : (
                  messages.map(msg => (
                    <div 
                      key={msg.id} 
                      className={`flex ${msg.senderId === user?.id ? 'justify-end' : 'justify-start'}`}
                    >
                      <div className={msg.senderId === user?.id ? 'chat-bubble-user' : 'chat-bubble-other'}>
                        <p>{msg.content}</p>
                        <div className="text-xs mt-1 opacity-70">
                          {new Date(msg.createdAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                        </div>
                      </div>
                    </div>
                  ))
                )}
                <div ref={messagesEndRef} />
              </div>

              {/* Message input */}
              <form onSubmit={sendMessage} className="p-4 border-t border-gray-200 flex">
                <input
                  type="text"
                  value={message}
                  onChange={(e) => setMessage(e.target.value)}
                  placeholder="Type a message..."
                  className="input"
                />
                <button 
                  type="submit"
                  className="ml-2 px-6 bg-primary hover:bg-indigo-700 text-white rounded-md"
                  disabled={!message.trim()}
                >
                  Send
                </button>
              </form>
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center flex-col p-4">
              <svg className="w-20 h-20 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"></path>
              </svg>
              <h2 className="text-xl font-medium text-gray-700 mb-2">Welcome to Chat</h2>
              <p className="text-gray-500 max-w-sm text-center">
                Select a chat from the sidebar or create a new buying or selling post to start chatting
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}