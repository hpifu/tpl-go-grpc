Feature: GetToken

    Scenario: echo
        When grpc 请求 echo Echo
            """
            {
                "rid": "12345678",
                "message": "hello world"
            }
            """
        Then grpc 检查
            """
            {
                "message": "hello world"
            }
            """
