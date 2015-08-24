import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;

import org.apache.commons.io.IOUtils;

public class echo {
    public static void main(String[] args) {
        if(args != null){
            for(String arg : args){
                System.out.println(arg);
            }

        }

        BufferedReader stdin = new BufferedReader(new InputStreamReader(
                System.in));
        String line = null;

        try {
            while ((line = stdin.readLine()) != null) {

                System.out.println(line);

            }
        } catch (IOException e) {
            IOUtils.closeQuietly(stdin);
        }
    }
}