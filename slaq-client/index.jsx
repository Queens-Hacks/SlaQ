'use strict';

// require("./node_modules/bootstrap/dist/css/bootstrap.min.css");
// require("./main.css");

import React from 'react';
import ReactDOM from 'react-dom';
// import Request from 'browser-request'

export class ChatBox extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      messages: [
        {
          name: "TestUser",
          id: -1,
          text: "FIRST MESSAGE",
          stars: 0
        }
      ],
      name: "Username",
      inputText: "",
      socket: null
    };
    this.componentDidMount = this.componentDidMount.bind(this);
    this.handleNewMessage = this.handleNewMessage.bind(this);
    this.handlePostMessage = this.handlePostMessage.bind(this);

    this.handleNameChange = this.handleNameChange.bind(this);
    this.handleInputTextChange = this.handleInputTextChange.bind(this);

  }
  handleNewMessage(messageInfo) {
    this.setState({messages: this.state.messages.concat(messageInfo)})

  }
  handlePostMessage(e) {
    if (e.keyCode === 13) {
      var outgoingMessage = {
        MessageText: this.state.inputText,
        MessageDisplayName: this.state.name
      };
      console.log(outgoingMessage)

      this.state.socket.send(JSON.stringify(outgoingMessage));
      this.setState({inputText: ""})
      return false
    }
  }
  handleNameChange(e) {
    this.setState({name: e.target.value});
  }
  handleInputTextChange(e) {
    this.setState({inputText: e.target.value});
  }
  componentDidMount() {
    this.state.socket = new WebSocket("ws://localhost:9999/ws/course/anycourse");

    this.state.socket.onmessage = (msg) => {
      let parsed = JSON.parse(msg.data)
      let payload = {
        name: parsed.MessageDisplayName,
        id: parsed.MessageId,
        text: parsed.MessageText,
        stars: 0
      }
      console.log(parsed, payload)
      this.handleNewMessage(payload)
    }

    this.state.socket.onclose = function(event) {
      console.log(closed)
    }
  }

  render() {
    return (
      <div className='MessageList'>
        <InputForm name ={this.state.name} inputText={this.state.inputText} handleNameChange={this.handleNameChange} handleInputTextChange={this.handleInputTextChange} handlePostMessage={this.handlePostMessage}/>
        <MessageList messages={this.state.messages}/>
      </div>
    )
  }
}

export class MessageList extends React.Component {
  constructor(props) {
    super(props);
  }
  render() {
    var messageNodes = this.props.messages.map(function(course) {
      return (
        <MessageCard data={course}></MessageCard>
      )
    })
    return (
      <ul id="messages">
        {messageNodes}
      </ul>
    )
  }
}

export class InputForm extends React.Component {
  constructor(props) {
    super(props);
  }

  render() {
    return (
      <form id="form">
        <label for="desiredName">Name</label>
        <input id="desiredName" value={this.props.name} type="text" onChange={this.props.handleNameChange}/>
        <label for="userMessage">Message</label>
        <input id="userMessage" value={this.props.inputText} type="text" onChange={this.props.handleInputTextChange} onKeyDown={this.props.handlePostMessage}/>
      </form>
    )
  }
}

export class MessageCard extends React.Component {
  constructor(props) {
    super(props);
  }
  render() {
    return (
      <li key ={this.props.data.id}>
        {this.props.data.stars}
        {this.props.data.name}
        {this.props.data.text}
      </li>
    )
  }
}
ReactDOM.render(
  <ChatBox url="http://159.203.112.6:3000"/>, document.querySelector("#myApp"));
